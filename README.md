1. Build docker image with raft example
```shell
docker build -t raft-example:latest .
```
2. Configure docker enginge to expose prometheus metrics
   https://docs.docker.com/config/daemon/prometheus/#configure-docker
3. Init docker swarm
```shell
docker swarm init
```
4. Deploy stack (example + promehtues)
```shell
docker stack deploy -c example-raft-stack.yaml raft
```
5. Set/get value 
```shell
curl -L http://127.0.0.1:8080/my-key -XPUT \
  -d '{ "value":"v1","previouslyObservedVersion":0 }' \
  -H 'Content-Type: application/json'
   
curl -L http://127.0.0.1:8080/my-key 
```

6. Logs
```shell
docker service logs raft_raft-example -f  
```
7. Locust
Go to http://localhost:8089
and start testing (example 10000 users, 100 spawn rate).

8. Metrics
- throughput and percentiles
```shell
http://localhost:9090/graph?g0.expr=label_replace(rate(app_set_value%5B5s%5D)%2C%20%22operation%22%2C%20%22set%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(rate(app_get_value%5B5s%5D)%2C%20%22operation%22%2C%20%22get%22%2C%20%22.*%22%2C%20%22.*%22)&g0.tab=0&g0.stacked=0&g0.show_exemplars=0&g0.range_input=5m&g1.expr=label_replace(histogram_quantile(0.01%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp01%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(0.05%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp05%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(0.3%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp30%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(0.5%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp50%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(0.99%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp90%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(0.01%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22getp100%22%2C%20%22.*%22%2C%20%22.*%22)%20or%20label_replace(histogram_quantile(1%2C%20sum(rate(app_get_duration_seconds_bucket%5B5s%5D))%20by%20(le))%2C%20%22operation%22%2C%20%22get-max%22%2C%20%22.*%22%2C%20%22.*%22)&g1.tab=0&g1.stacked=0&g1.show_exemplars=0&g1.range_input=5m
```
9. Grafana with metrics
   http://localhost:8085/?orgId=1
