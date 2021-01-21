# TCPB

TCP Bridge under websocket protocol.

> implmented with go websocket [gorilla/websocket](https://github.com/gorilla/websocket)


## Usage

run in tunnel server:

```bash
./server -port 4223

# using docker
docker run wuhuizuo/tcpb -v <host_port>:80
```

run in user client side:

```bash
./client -tunnel ws://{tunnel server ip}:4223/{remote_tcp_ip}:{remote_tcp_port} -port {local_port}
```



## test

run a tcp server on remote server on :

```bash
./tcp_server.py
```

file `tcp_server.py`, as remote `{remote_tcp_port}` is `65432`:
```python
#! /usr/bin/env python3
# a simple tcp server
import socket,os
sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
sock.bind(('0.0.0.0', 65432))
sock.listen(5)
while True:
    connection,address = sock.accept()
    print("accept connection from:", address)
    buf = connection.recv(1024)
    print(buf)
    connection.send(buf)
    connection.close()

```

test tunnel with tcp client:

```bash
./tcp_client.py
```

file `tcp_server.py`, as tunneled `{local_port}` is `65431`:
```python
#!/usr/bin/env python3

import socket

HOST = '127.0.0.1'  # The server's hostname or IP address
PORT = 65432        # The port used by the server

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
    s.connect((HOST, PORT))
    s.sendall(b'Hello, world')
    print("send ok")
    data = s.recv(1024)

print('Received', repr(data))
```