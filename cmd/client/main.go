package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/wuhuizuo/tcpb"
)

func handleConnection(c net.Conn, wsURL string, messageType int) {
	defer c.Close()

	bridge := tcpb.Bridge{PackType: messageType}
	if err := bridge.TCP2WS(c, wsURL); err != nil {
		log.Println("[ERROR] ", err)
	}
}


func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <tcpTargetAddress>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var host string
	var port uint
	var tunnelURL string
	var certFile string
	var keyFile string
	var messageType int

	flag.StringVar(&host, "host", "", "The ip to bind on, default all")
	flag.UintVar(&port, "p", 4223, "The port to listen on")
	flag.UintVar(&port, "port", 4223, "The port to listen on")
	flag.StringVar(&tunnelURL, "u", "", "tunnel url")
	flag.StringVar(&tunnelURL, "url", "", "tunnel url")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path")
	flag.IntVar(&messageType, "m", 1, "msg frame type, 1:text, 2: binary")
	flag.IntVar(&messageType, "msgtype", 1, "msg frame type, 1:text, 2: binary")
	flag.Usage = usage
	flag.Parse()

	serveAddr := fmt.Sprintf("%s:%d", host, port)
	log.Println("[INFO] message type:", messageType)
	log.Println("[INFO] TCP tunneled on ", serveAddr)
	l, err := net.Listen("tcp", serveAddr)
	if err != nil {
        log.Fatal(err)
    }
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go handleConnection(c, tunnelURL, messageType)
	}
}


