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

func relayHandler(upgrader *websocket.Upgrader) httpHandlerFunc {
	if upgrader == nil {
		upgrader = &websocket.Upgrader{}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		tcpAddress := strings.TrimLeft(r.URL.Path, "/")
		if tcpAddress == "" {
			w.Header().Add("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("empty remote address"))
			return
		}

		log.Println("[INFO ] receive tunnel request for tcp: ", tcpAddress)
		wsCon, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR] %+v\n", err)
			return
		}
		defer func() {
			log.Println("[INFO ] close websocket connection")
			wsCon.Close()
		}()

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

	flag.StringVar(&host, "host", "", "The ip to bind on, default all")
	flag.UintVar(&port, "port", 8080, "The port to listen on")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path")
	flag.Usage = usage
	flag.Parse()

	http.HandleFunc("/", relayHandler(nil))

	serveAddr := fmt.Sprintf("%s:%d", host, port)

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
