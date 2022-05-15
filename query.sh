#!/usr/bin/env bash
set -x

curl -s -L http://127.0.0.1:8080/my-key -XPUT \
  -d '{ "value":"v1","previouslyObservedVersion":0 }' \
  -H 'Content-Type: application/json' \
  | jq

curl -s -L http://127.0.0.1:8080/my-key \
  -H 'Content-Type: application/json' | jq

# two conflicting writes
curl -s -L http://127.0.0.1:8080/my-key -XPUT \
  -d '{ "value":"v2","previouslyObservedVersion":1 }' \
  -H 'Content-Type: application/json' | jq

curl -s -L http://127.0.0.1:8080/my-key -XPUT \
  -d '{ "value":"v2-concurrent","previouslyObservedVersion":1 }' \
  -H 'Content-Type: application/json' | jq

curl -s -L http://127.0.0.1:8080/my-key | jq
