package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/wuhuizuo/tcpb"
)

// 代理类型
const (
	proxyNone = "noProxy"
	proxyEnv  = "env"
)

type proxyGetter func(*http.Request) (*url.URL, error)

func main() {
	var host string
	var port uint
	var tunnelURL string
	var proxyURL string
	var certFile string
	var keyFile string

	flag.StringVar(&host, "host", "", "The ip to bind on, default all")
	flag.UintVar(&port, "port", 0, "The port to listen on, default automatically chosen.")
	flag.StringVar(&tunnelURL, "tunnel", "", "tunnel url, format: ws://host:port")
	flag.StringVar(&proxyURL, "proxy", "", "proxy url, format: http[s]://host:port/path, default use system proxy.")
	flag.StringVar(&certFile, "tlscert", "", "TLS cert file path.")
	flag.StringVar(&keyFile, "tlskey", "", "TLS key file path.")
	flag.Usage = usage
	flag.Parse()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		oscall := <-c
		log.Printf("[WARN ] system call:%+v", oscall)
		cancel()
	}()

	serveAddr := fmt.Sprintf("%s:%d", host, port)

	if err := serve(ctx, serveAddr, tunnelURL, proxyURL); err != nil {
		log.Printf("[ERROR] failed to serve:+%v\n", err)
		os.Exit(1)
	}
}


func serve(ctx context.Context, serveAddr, tunnelURL, proxyURL string) error {
	l, err := net.Listen("tcp", serveAddr)
	if err != nil {
		return err
	}

	log.Printf("[INFO ] TCP tunnel started on %s [%s]\n", l.Addr().String(), l.Addr().Network())
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				log.Fatalf("[ERROR] accept connection failed: %+v\n", err)
			}
			log.Printf("[INFO ] accepted connection %s -> %s\n", c.RemoteAddr(), c.LocalAddr())

			go handleConnection(c, tunnelURL, getWSProxy(proxyURL))
		}
	}()

	<-ctx.Done()
	log.Printf("[INFO ] TCP tunnel stopping on %s [%s]\n", l.Addr().String(), l.Addr().Network())

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = l.Close(); err != nil {
		return errors.Wrap(err, "server Shutdown Failed")
	}

	log.Println("[INFO ] TCP tunnel stopped")
	return nil
}

func handleConnection(c net.Conn, wsURL string, wsProxyGetter proxyGetter) {
	defer func() {
		log.Println("[INFO ] close client connection")
		c.Close()
	}()

	bridge := tcpb.Bridge{WSProxyGetter: wsProxyGetter, HeartInterval: 30 * time.Second}
	err := bridge.TCP2WS(c, wsURL)
	if err != nil {
		log.Printf("[ERROR] %+v\n", err)
	}		
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options], options list:\n", os.Args[0])
	flag.PrintDefaults()
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
