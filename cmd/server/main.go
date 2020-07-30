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

		if tcpAddress == "" {
			log.Println("[ERROR] ", "url path is not tcp address")
			return
		}

		wsCon, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}

		bridge := tcpb.Bridge{PackType: messageType}
		if err := bridge.WS2TCP(wsCon, tcpAddress); err != nil {
			log.Println("[ERROR] ", err)
		}
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <tcpTargetAddress>\n", os.Args[0])
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
	log.Println("[INFO] message type:", messageType)
	log.Printf("[INFO] Listening on %s:%d\n", host, port)

	var err error
	if certFile == "" || keyFile == "" {
		err = http.ListenAndServe(serveAddr, nil)
	} else {
		err = http.ListenAndServeTLS(serveAddr, certFile, keyFile, nil)
	}

	if err != nil {
		log.Fatal(err)
	}
}
