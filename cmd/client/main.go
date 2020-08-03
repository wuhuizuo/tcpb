package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/wuhuizuo/tcpb"
)

// 代理类型
const (
	proxyNone = "noProxy"
	proxyEnv  = "env"
)

type proxyGetter func(*http.Request) (*url.URL, error)

func handleConnection(c net.Conn, wsURL string, wsProxyGetter proxyGetter) {
	defer c.Close()

	bridge := tcpb.Bridge{WSProxyGetter: wsProxyGetter}
	err := bridge.TCP2WS(c, wsURL)
	if err != nil {
		log.Printf("[ERROR] %+v\n", err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options], options list:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var host string
	var port uint
	var tunnelURL string
	var proxyURL string
	var certFile string
	var keyFile string
	var messageType int

	flag.StringVar(&host, "host", "", "The ip to bind on, default all")
	flag.UintVar(&port, "port", 0, "The port to listen on, default automatically chosen.")
	flag.StringVar(&tunnelURL, "tunnel", "", "tunnel url, format: ws://host:port")
	flag.StringVar(&proxyURL, "proxy", "", "proxy url, format: http[s]://host:port/path, default use system proxy.")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path.")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path.")
	flag.IntVar(&messageType, "msgtype", 1, "msg frame type, 1:text, 2: binary.")
	flag.Usage = usage
	flag.Parse()

	serveAddr := fmt.Sprintf("%s:%d", host, port)
	log.Println("[INFO ] message type:", messageType)
	l, err := net.Listen("tcp", serveAddr)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()
	log.Printf("[INFO ] TCP tunneled on %s [%s]\n", l.Addr().String(), l.Addr().Network())

	for {
		c, err := l.Accept()
		if err != nil {
			log.Fatalf("[ERROR] accept connection failed: %+v\n", err)
		}
		log.Printf("[INFO ] accepted connection %s -> %s\n", c.RemoteAddr(), c.LocalAddr())

		go handleConnection(c, tunnelURL, getWSProxy(proxyURL))
	}
}

func getWSProxy(proxy string) proxyGetter {
	switch proxy {
	case proxyNone:
		return nil
	case proxyEnv, "":
		return http.ProxyFromEnvironment
	default:
		proxyURI, err := url.ParseRequestURI(proxy)
		if err != nil {
			log.Fatalln("[ERROR] proxy url invalid", err)
		}
		return http.ProxyURL(proxyURI)
	}
}

func init() {
	log.SetOutput(os.Stdout)
}
