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
	"encoding/json"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.etcd.io/etcd/raft/v3/raftpb"
	"io"
	"log"
	"net/http"
	"strconv"
)

// Handler for a http based key-value store backed by raft
type httpKVAPI struct {
	store       *kvstore
	confChangeC chan<- raftpb.ConfChange
}

var (
	appSetValue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_set_value",
	})
	appSetValueFail = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_set_value_fail",
	})
	appGetValue = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_get_value",
	})
	appGetValueFail = promauto.NewCounter(prometheus.CounterOpts{
		Name: "app_get_value_fail",
	})
	setDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "app_set_duration_seconds",
		Buckets: prometheus.LinearBuckets(0.0001, 0.0001, 100),
	})
	getDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "app_get_duration_seconds",
		Buckets: prometheus.LinearBuckets(0.0001, 0.0001, 100),
	})
)

type putRequest struct {
	Value                     string `json:"value"`
	PreviouslyObservedVersion int    `json:"previouslyObservedVersion"`
}

type PutRequestResponse struct {
	Success        bool   `json:"success"`
	Key            string `json:"key"`
	CurrentValue   string `json:"value"`
	CurrentVersion int    `json:"version"`
}

func (h *httpKVAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.RequestURI
	defer r.Body.Close()
	switch r.Method {
	case http.MethodPut:
		timer := prometheus.NewTimer(setDuration)
		defer timer.ObserveDuration()
		v, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on PUT (%v)\n", err)
			appSetValueFail.Inc()
			http.Error(w, "Failed to read request body.", http.StatusBadRequest)
			return
		}

		var request putRequest
		err = json.Unmarshal(v, &request)
		if err != nil {
			log.Printf("Failed to deserialize json on PUT (%v)\n", err)
			http.Error(w, "Invalid request body.", http.StatusBadRequest)
			return
		}

		value := request.Value
		prevVersion := request.PreviouslyObservedVersion

		result, _ := h.store.Propose(key, value, prevVersion)
		if result.Success {
			appSetValue.Inc()
		} else {
			appSetValueFail.Inc()
		}

		response := PutRequestResponse{
			Key:            key,
			CurrentValue:   result.CurrentValue.Val,
			CurrentVersion: result.CurrentValue.Version,
			Success:        result.Success,
		}

		_ = json.NewEncoder(w).Encode(response)
		w.WriteHeader(http.StatusOK)

	case http.MethodGet:
		timer := prometheus.NewTimer(getDuration)
		defer timer.ObserveDuration()
		if v, ok := h.store.Lookup(key); ok {
			bytes, _ := json.Marshal(v)
			_, _ = w.Write(bytes)
			appGetValue.Inc()
		} else {
			appGetValueFail.Inc()
			http.Error(w, "Failed to GET", http.StatusNotFound)
		}
	case http.MethodPost:
		url, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Failed to read on POST (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		nodeId, err := strconv.ParseUint(key[1:], 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on POST", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:    raftpb.ConfChangeAddNode,
			NodeID:  nodeId,
			Context: url,
		}
		h.confChangeC <- cc
		// As above, optimistic that raft will apply the conf change
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		nodeId, err := strconv.ParseUint(key[1:], 0, 64)
		if err != nil {
			log.Printf("Failed to convert ID for conf change (%v)\n", err)
			http.Error(w, "Failed on DELETE", http.StatusBadRequest)
			return
		}

		cc := raftpb.ConfChange{
			Type:   raftpb.ConfChangeRemoveNode,
			NodeID: nodeId,
		}
		h.confChangeC <- cc

		// As above, optimistic that raft will apply the conf change
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", http.MethodPut)
		w.Header().Add("Allow", http.MethodGet)
		w.Header().Add("Allow", http.MethodPost)
		w.Header().Add("Allow", http.MethodDelete)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// serveHttpKVAPI starts a key-value server with a GET/PUT API and listens.
func serveHttpKVAPI(kv *kvstore, port int, confChangeC chan<- raftpb.ConfChange, errorC <-chan error) {
	http.Handle("/", &httpKVAPI{
		store:       kv,
		confChangeC: confChangeC,
	})
	http.Handle("/metrics", promhttp.Handler())
	_ = http.ListenAndServe(":"+strconv.Itoa(port), nil)

	// exit when raft goes down
	if err, ok := <-errorC; ok {
		log.Fatal(err)
	}
}
