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

// proxy types
const (
	proxyNone = "noProxy"
	proxyEnv  = "env"
)

// release version info
var (
	version   string = "unknown"
	buildDate string = "unknown"
)

type (
	proxyGetter func(*http.Request) (*url.URL, error)

	clientTunnelCfg struct {
		tunnelURL         string
		proxyURL          string
		heartbeatInterval uint
		userInfo          *tcpb.HTTPBasicUserInfo
	}

	clientCfg struct {
		clientTunnelCfg

		listenHost string
		listenPort uint
	}
)

func main() {
	config := parseCmdArgs()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		oscall := <-c
		log.Printf("[WARN ] system call:%+v", oscall)
		cancel()
	}()

	if err := serve(ctx, config); err != nil {
		log.Printf("[ERROR] failed to serve:+%v\n", err)
		os.Exit(1)
	}
}

func parseCmdArgs() clientCfg {
	var config clientCfg
	flag.StringVar(&config.listenHost, "host", "", "The ip to bind on, default all")
	flag.UintVar(&config.listenPort, "port", 0, "The port to listen on, default automatically chosen.")
	flag.UintVar(&config.heartbeatInterval, "heartbeat", 30, "The interval(second) for heartbeat sending to tunnel server.")
	flag.StringVar(&config.tunnelURL, "tunnel", "", "tunnel url, format: ws://host:port")
	flag.StringVar(&config.proxyURL, "proxy", "", "proxy url, format: http[s]://host:port/path, default use system proxy.")

	tunnelAuthUsername := flag.String("user", "", "tunnel user name.")
	tunnelAuthPassword := flag.String("password", "", "tunnel password.")
	showVersion := flag.Bool("version", false, "prints current version")
	flag.Usage = usage

	flag.Parse()

	if showVersion != nil && *showVersion {
		printVersion()
		os.Exit(0)
	}

	// detect user:password info in tunnelURL
	u, err := url.ParseRequestURI(config.tunnelURL)
	if err != nil {
		log.Printf("[ERROR] invalid tunnel url: %s\n", config.tunnelURL)
		os.Exit(2)
	}

	// user,password set in url.
	if u.User != nil {
		password, _ := u.User.Password()
		config.userInfo = &tcpb.HTTPBasicUserInfo{
			Username: u.User.Username(),
			Password: password,
		}
	}

	// user,password set in cmd args.
	if tunnelAuthUsername != nil {
		config.userInfo = &tcpb.HTTPBasicUserInfo{
			Username: *tunnelAuthUsername,
		}
		if tunnelAuthPassword != nil {
			config.userInfo.Password = *tunnelAuthPassword
		}
	}

	return config
}

func serve(ctx context.Context, cfg clientCfg) error {
	serveAddr := fmt.Sprintf("%s:%d", cfg.listenHost, cfg.listenPort)
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
		return errors.Wrap(err, "server Shutdown Failed")
	}

	log.Println("[INFO ] TCP tunnel stopped")
	return nil
}

func handleConnection(c net.Conn, tunnelCfg clientTunnelCfg) {
	defer func() {
		log.Println("[INFO ] close client connection")
		c.Close()
	}()

	bridge := tcpb.Bridge{
		WSProxyGetter: getWSProxy(tunnelCfg.proxyURL),
		HeartInterval: time.Duration(tunnelCfg.heartbeatInterval) * time.Second,
	}
	var userInfo tcpb.HTTPUserInfo
	if tunnelCfg.userInfo != nil {
		userInfo = tunnelCfg.userInfo
	}

	err := bridge.TCP2WS(c, tunnelCfg.tunnelURL, userInfo)
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
