# TCPB

TCP Bridge under HTTP protocol.

> websocket tunnel side/client implmented with go websocket [gorilla/websocket](https://github.com/gorilla/websocket)
> http connect/post tunnel server directly using envoy, only implmented tunnel client in my repo.

## Usage

### start up tcp server and envoy tunnel server side

```bash
docker-compose -f .docker/docker-compose.yml up -d
```

### start tunnel client side

```bash
# tunnel remote tcp to local port 10001

# http post tunnel, using config in docker-compose.yaml: /etc/envoy/envoy_post.yaml
go run ./cmd/client/ --tunnel=http://127.0.0.1:10000 -port 10001 --method=POST

# http connect tunnel, using config in docker-compose.yaml: /etc/envoy/envoy_connect.yaml
# go run ./cmd/client/ --tunnel=http://127.0.0.1:10000 -port 10001
```

### test with tcp client

test envoy encapsulate tcp server：

```bash
# type STOP to exist.
go run ./cmd/test-tool --addr 127.0.0.1:20000
```

test with tunnel client function:

```bash
# type STOP to exist.
go run ./cmd/test-tool --addr 127.0.0.1:10001
```

test websocket client/side:

```bash
# terminal 1
go run ./cmd/server/ -port 30000
# terminal 2
go run ./cmd/client/ --tunnel=ws://127.0.0.1:30000/127.0.0.1:20000 -port 10001
# terminal 3
go run ./cmd/test-tool --addr 127.0.0.1:10001

```
