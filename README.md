1. Build docker image with raft example
```shell
docker build -t raft-example .
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
curl -L http://127.0.0.1:8080/my-key -XPUT -d bar

curl -L http://127.0.0.1:8080/my-key 
```

6. Logs
```shell
docker service logs raft_raft-example -f  
```

