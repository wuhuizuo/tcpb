package main

import (
	"net/http"
	"net/url"
)

type proxyGetter func(*http.Request) (*url.URL, error)

type clientCfg struct {
	clientTunnelCfg

	listenHost string
	listenPort uint
}

type clientTunnelCfg struct {
	tunnelURL  string
	httpMethod string

	proxyURL string

	heartbeatInterval uint
}
