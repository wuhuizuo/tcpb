TARGETS = tcpbc tcpbs

all: clean client server

client:
	go build -o tcpbc ./cmd/client

server: 
	go build -o tcpbs ./cmd/server

clean: 
	rm -rf $(TARGETS)
	
