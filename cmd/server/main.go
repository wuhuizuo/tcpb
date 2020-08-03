package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/wuhuizuo/tcpb"
)

type httpHandlerFunc func(w http.ResponseWriter, r *http.Request)

func relayHandler(messageType int, upgrader *websocket.Upgrader) httpHandlerFunc {
	if upgrader == nil {
		upgrader = &websocket.Upgrader{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tcpAddress := strings.TrimLeft(r.URL.Path, "/")
		log.Println("[INFO ] receive tunnel request for tcp: ", tcpAddress)

		if tcpAddress == "" {
			log.Println("[ERROR] ", "url path is not tcp address")
			return
		}

		wsCon, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR] %+v\n", err)
			return
		}

		bridge := tcpb.Bridge{}
		if err := bridge.WS2TCP(wsCon, tcpAddress); err != nil {
			log.Printf("[ERROR] %+v\n", err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options], options list:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var host string
	var port uint
	var certFile string
	var keyFile string
	var messageType int

	flag.StringVar(&host, "host", "", "The ip to bind on, default all")
	flag.UintVar(&port, "p", 4223, "The port to listen on")
	flag.UintVar(&port, "port", 4223, "The port to listen on")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path")
	flag.IntVar(&messageType, "m", 1, "msg frame type, 1:text, 2: binary")
	flag.IntVar(&messageType, "msgtype", 1, "msg frame type, 1:text, 2: binary")
	flag.Usage = usage
	flag.Parse()

	http.HandleFunc("/", relayHandler(messageType, nil))

	serveAddr := fmt.Sprintf("%s:%d", host, port)
	log.Println("[INFO ] message type:", messageType)

	var err error
	if certFile == "" || keyFile == "" {
		log.Printf("[INFO ] Listening on ws://%s\n", serveAddr)
		err = http.ListenAndServe(serveAddr, nil)
	} else {
		log.Printf("[INFO ] Listening on wss://%s\n", serveAddr)
		err = http.ListenAndServeTLS(serveAddr, certFile, keyFile, nil)
	}

	if err != nil {
		log.Fatalf("[ERROR] %+v\n", err)
	}
}

func init() {
	log.SetOutput(os.Stdout)
}
