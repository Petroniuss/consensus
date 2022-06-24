#!/usr/bin/env bash
duration_in_sec=120
warmup_in_sec=30
cooldown_in_sec=30

get_requests=0
conflict_requests=0
success_requests=0
update_counters=false
end_time=$(date -u -v+"$((warmup_in_sec + duration_in_sec + cooldown_in_sec))"S +%s)
start_collect_time=$(date -u -v+"$((warmup_in_sec))"S +%s)
stop_collect_time=$(date -u -v+"$((warmup_in_sec + duration_in_sec))"S +%s)

previous_version=$(curl -s -L http://127.0.0.1:8080/my-key \
  -H 'Content-Type: application/json' | jq '.version')

while [[ $(date -u +%s) -le $end_time ]]; do

  random_perc=$(($RANDOM % 100))
  option=0

  if [[ $random_perc -lt 50 ]]; then
    option=0
  elif [[ $random_perc -lt 90 ]]; then
    option=1
  else
    option=2
  fi
  echo $option
  echo $option
  if [[ $(date -u +%s) -ge $start_collect_time ]]; then
    update_counters=true
  fi
  if [[ $(date -u +%s) -ge $stop_collect_time ]]; then
    update_counters=false
  fi

  case $option in
  0)
    previous_version=$(curl -s -L http://127.0.0.1:8080/my-key \
      -H 'Content-Type: application/json' | jq '.version')

    if [ "$update_counters" = true ]; then
      echo "collect"
      get_requests=$((get_requests + 1))
    fi
    ;;
  1)
    previous_version=$(curl -s -L http://127.0.0.1:8080/my-key -XPUT \
      -d "{ \"value\":\"v1\",\"previouslyObservedVersion\":$previous_version }" \
      -H 'Content-Type: application/json' |
      jq '.version')

    if [ "$update_counters" = true ]; then
      echo "collect"
      success_requests=$((success_requests + 1))
    fi
    ;;
  2)
    previous_version=$(curl -s -L http://127.0.0.1:8080/my-key -XPUT \
      -d '{ "value":"v1","previouslyObservedVersion":0 }' \
      -H 'Content-Type: application/json' |
      jq '.version')
    if [ "$update_counters" = true ]; then
      echo "collect"
      conflict_requests=$((conflict_requests + 1))
    fi
    ;;
  *) echo -n "unknown" ;;
  esac
  sleep 0.1
done
echo "Time: $(date -r $start_collect_time) - $(date -r $stop_collect_time)"
echo "Success counter: $success_requests"
echo "Conflict counter: $conflict_requests"
echo "Get counter: $get_requests"
