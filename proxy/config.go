package proxy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// time duration consts.
const (
	DefaultDiaTimeout      = 1 * time.Second
	DefaultWSHeartInterval = 30 * time.Second
)

// supported http schemes
var supportedSchemes = map[string]bool{
	SchemeHTTP:         true,
	SchemeHTTPS:        true,
	SchemeWebsocket:    true,
	SchemeWebsocketSec: true,
}

// Config for proxy.
type Config struct {
	Base            BaseConfig
	WSHeartInterval time.Duration // websocket part: interval for websocket send ping package to keep alive.

	HTTPMethod string // http part: which method to use for dialing, default: CONNECT.
}

// BaseConfig for proxy.
type BaseConfig struct {
	TLSClientConfig *tls.Config   // tls client config for https|wss
	Header          http.Header   // http addon header
	DialTimeout     time.Duration // proxy dial timeout
}

// DefaultConfig return default config for common using, tls will skip tls verify.
func DefaultConfig(u *url.URL) *Config {
	ret := &Config{
		Base: BaseConfig{
			DialTimeout: DefaultDiaTimeout,
		},
		WSHeartInterval: DefaultWSHeartInterval,
		HTTPMethod:      http.MethodConnect,
	}

	// set tls.
	switch u.Scheme {
	case SchemeHTTPS, SchemeWebsocketSec:
		ret.Base.TLSClientConfig = &tls.Config{
			ServerName:         u.Hostname(),
			InsecureSkipVerify: true,
		}
	default:
		// nothing
	}

	return ret
}
