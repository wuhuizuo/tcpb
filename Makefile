TARGETS = tcpbc tcpbs

client:
	go build -o tcpbc ./cmd/client

server: 
	go build -o tcpbs ./cmd/server

clean: 
	rm -rf $(TARGETS)
	
all: clean client server