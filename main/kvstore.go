// Copyright 2015 The etcd Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"go.etcd.io/etcd/pkg/v3/idutil"
	"go.etcd.io/etcd/pkg/v3/wait"
	"log"
	"sync"
	"time"

	"go.etcd.io/etcd/raft/v3/raftpb"
	"go.etcd.io/etcd/server/v3/etcdserver/api/snap"
)

// RaftPutRequest is sent through raft and serialized using glob.
type RaftPutRequest struct {
	Key                       string
	Value                     string
	PreviouslyObservedVersion int
	ReqId                     uint64
}

// a key-value store backed by raft
type kvstore struct {
	proposeC    chan<- string // channel for proposing updates
	commitC     <-chan *commit
	mu          sync.RWMutex
	kvStore     map[string]VersionedValue // current committed key-value pairs
	snapshotter *snap.Snapshotter
	wait        wait.Wait
	reqIDGen    *idutil.Generator
}

type VersionedValue struct {
	Val     string `json:"value"`
	Version int    `json:"version"`
}

func newKVStore(snapshotter *snap.Snapshotter, proposeC chan<- string, commitC <-chan *commit, errorC <-chan error,
	nodeId int) *kvstore {
	s := &kvstore{
		proposeC:    proposeC,
		commitC:     commitC,
		kvStore:     make(map[string]VersionedValue),
		snapshotter: snapshotter,
		wait:        wait.New(),
		reqIDGen:    idutil.NewGenerator(uint16(nodeId), time.Now()),
	}

	snapshot, err := s.loadSnapshot()
	if err != nil {
		log.Panic(err)
	}
	if snapshot != nil {
		log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
		if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
			log.Panic(err)
		}
	}
	// read commits from raft into kvStore map until error
	go s.readCommits(commitC, errorC)
	return s
}

func (s *kvstore) Lookup(key string) (VersionedValue, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.kvStore[key]
	return v, ok
}

func (s *kvstore) Propose(key string, value string, previouslyObservedVersion int) (AppliedPutRequestResult, error) {
	var buf bytes.Buffer

	reqID := s.reqIDGen.Next()
	putRequest := RaftPutRequest{
		Key:                       key,
		Value:                     value,
		PreviouslyObservedVersion: previouslyObservedVersion,
		ReqId:                     reqID,
	}

	if err := gob.NewEncoder(&buf).Encode(putRequest); err != nil {
		log.Fatal(err)
	}

	// propose a value
	s.proposeC <- buf.String()

	// register a channel indicating that value has been acknowledged by a raft module.
	waitingChan := s.wait.Register(reqID)

	// a decent implementation would use some sort of timeout here,
	// but that would require a bit more refactoring.
	// see how it's done in etcd/v3_server.go.
	select {
	case x := <-waitingChan:
		return x.(AppliedPutRequestResult), nil
	}
}

type AppliedPutRequestResult struct {
	CurrentValue VersionedValue
	Success      bool
}

func (s *kvstore) readCommits(commitC <-chan *commit, errorC <-chan error) {
	for commit := range commitC {
		if commit == nil {
			// signaled to load snapshot
			snapshot, err := s.loadSnapshot()
			if err != nil {
				log.Panic(err)
			}
			if snapshot != nil {
				log.Printf("loading snapshot at term %d and index %d", snapshot.Metadata.Term, snapshot.Metadata.Index)
				if err := s.recoverFromSnapshot(snapshot.Data); err != nil {
					log.Panic(err)
				}
			}
			continue
		}

		for _, data := range commit.data {
			var putRequest RaftPutRequest
			dec := gob.NewDecoder(bytes.NewBufferString(data))
			if err := dec.Decode(&putRequest); err != nil {
				log.Fatalf("raftexample: could not decode message (%v)", err)
			}

			s.mu.Lock()

			key := putRequest.Key
			value, ok := s.kvStore[key]
			if !ok || value.Version == putRequest.PreviouslyObservedVersion {
				newValue := VersionedValue{
					Val:     putRequest.Value,
					Version: putRequest.PreviouslyObservedVersion + 1,
				}

				s.kvStore[key] = newValue

				log.Printf("RaftPutRequest (id: %d): applied succesfully.", putRequest.ReqId)
				s.wait.Trigger(putRequest.ReqId, AppliedPutRequestResult{
					CurrentValue: newValue,
					Success:      true,
				})
			} else {
				log.Printf("RaftPutRequest (id: %d): rejected.", putRequest.ReqId)
				oldValue := s.kvStore[key]
				s.wait.Trigger(putRequest.ReqId, AppliedPutRequestResult{
					CurrentValue: oldValue,
					Success:      false,
				})
			}

			s.mu.Unlock()
		}
		close(commit.applyDoneC)
	}
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}

func (s *kvstore) getSnapshot() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return json.Marshal(s.kvStore)
}

func (s *kvstore) loadSnapshot() (*raftpb.Snapshot, error) {
	snapshot, err := s.snapshotter.Load()
	if err == snap.ErrNoSnapshot {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *kvstore) recoverFromSnapshot(snapshot []byte) error {
	var store map[string]VersionedValue
	if err := json.Unmarshal(snapshot, &store); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.kvStore = store
	return nil
}
