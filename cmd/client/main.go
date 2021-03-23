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
	"github.com/wuhuizuo/tcpb/proxy"
)

// proxy types
const (
	proxyNone = "noProxy"
	proxyEnv  = "env"

	errCodeArgInvalid = -2
)

// release version info
var (
	version   string = "unknown"
	buildDate string = "unknown"
)

func main() {
	config, err := parseCmdArgs()
	if err != nil {
		log.Printf("[ERROR] %s", err)
		os.Exit(errCodeArgInvalid)
	}
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		oscall := <-c
		log.Printf("[WARN ] system call:%+v", oscall)
		cancel()
	}()

	if err := serve(ctx, *config); err != nil {
		log.Printf("[ERROR] failed to serve:+%v\n", err)
		os.Exit(1)
	}

	log.Println("[WARN ] exit normally.")
	os.Exit(0)
}

func parseCmdArgs() (*clientCfg, error) {
	var config clientCfg
	flag.StringVar(&config.listenHost, "host", "", "The ip to bind on, default all")
	flag.UintVar(&config.listenPort, "port", 0, "The port to listen on, default automatically chosen.")
	flag.UintVar(&config.heartbeatInterval, "heartbeat", 30, "The interval(second) for heartbeat sending to tunnel server.")
	flag.StringVar(&config.tunnelURL, "tunnel", "", "tunnel url, format: (ws|http|https)://[user:name@]host:port[/path]")
	flag.StringVar(&config.proxyURL, "proxy", "", "proxy url, format: http[s]://[user:name@]host:port[/path], default use system proxy.")
	flag.StringVar(&config.httpMethod, "method", http.MethodPost, "http proxy method: POST|CONNECT, only for http/https tunnel url")

	showVersion := flag.Bool("version", false, "prints current version")
	flag.Usage = usage

	flag.Parse()

	if showVersion != nil && *showVersion {
		printVersion()
		os.Exit(0)
	}

	return &config, nil
}

func serve(ctx context.Context, cfg clientCfg) error {
	if err := registryProxy(cfg); err != nil {
		return errors.WithStack(err)
	}

	l, err := net.Listen("tcp", net.JoinHostPort(cfg.listenHost, fmt.Sprint(cfg.listenPort)))
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

			go handleConnection(c, cfg.clientTunnelCfg)
		}
	}()

	<-ctx.Done()
	log.Printf("[INFO ] TCP tunnel stopping on %s [%s]\n", l.Addr().String(), l.Addr().Network())

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err = l.Close(); err != nil {
		log.Printf("[ERROR] %s\n", err)
		return errors.Wrap(err, "server shutdown failed.")
	}

	log.Println("[INFO ] TCP tunnel stopped.")
	return nil
}

// registry tcp proxy on http dialer.
func registryProxy(cfg clientCfg) error {
	tunnelURI, err := url.Parse(cfg.tunnelURL)
	if err != nil {
		return errors.WithStack(err)
	}

	proxyCfg := proxy.DefaultConfig(tunnelURI)
	proxyCfg.HTTPMethod = cfg.httpMethod
	proxyCfg.WSHeartInterval = time.Duration(cfg.heartbeatInterval) * time.Second

	proxy.RegisterProxyDialer(proxyCfg)

	return nil
}

func handleConnection(c net.Conn, tunnelCfg clientTunnelCfg) {
	defer func() {
		log.Printf("[WARN ] close client tcp connection: %s -> %s \n", c.LocalAddr(), c.RemoteAddr())
		c.Close()
	}()

	bridge := tcpb.Bridge{
		WSProxyGetter: getWSProxy(tunnelCfg.proxyURL),
		HeartInterval: time.Duration(tunnelCfg.heartbeatInterval) * time.Second,
	}

	err := bridge.TCP2Tunnel(c, tunnelCfg.tunnelURL)
	if err != nil {
		log.Printf("[ERROR] %+v\n", err)
	}
}

func printVersion() {
	fmt.Fprintln(os.Stdout, "Version:\t", version)
	fmt.Fprintln(os.Stdout, "Build date:\t", buildDate)
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
