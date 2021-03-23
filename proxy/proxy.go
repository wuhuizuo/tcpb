package proxy

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"

	"github.com/wuhuizuo/tcpb/proxy/base"
	"github.com/wuhuizuo/tcpb/proxy/connect"
	"github.com/wuhuizuo/tcpb/proxy/internal"
	"github.com/wuhuizuo/tcpb/proxy/post"
	"github.com/wuhuizuo/tcpb/proxy/websocket"

	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

// Schemes
const (
	SchemeHTTP         = "http"
	SchemeHTTPS        = "https"
	SchemeWebsocket    = "ws"
	SchemeWebsocketSec = "wss"
)

// RegisterProxyDialer register proxy dialer for http/https/ws/wss schemes.
func RegisterProxyDialer(proxyConfig *Config) {
	proxy.RegisterDialerType(SchemeHTTP, GeneratorWithConfig(proxyConfig))
	proxy.RegisterDialerType(SchemeHTTPS, GeneratorWithConfig(proxyConfig))
	proxy.RegisterDialerType(SchemeWebsocket, GeneratorWithConfig(proxyConfig))
	proxy.RegisterDialerType(SchemeWebsocketSec, GeneratorWithConfig(proxyConfig))
}

// GeneratorWithConfig is like ConnectFromURLWithConfig, but is suitable for passing to
// proxy.RegisterDialerType while maintaining configuration options.
//
// This is to enable registration of an http(s) proxy with options, e.g.:
//     proxy.RegisterDialerType("https", myproxy.GeneratorWithConfig(
//             &myproxy.Config{DialTimeout: 5 * time.Minute},
//     ))
func GeneratorWithConfig(cfg *Config) base.SchemeDialerGenerator {
	return func(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
		return FromURLWithConfig(u, forward, cfg)
	}
}

// FromURLBy returns a proxy.Dialer given a URL specification and an underlying
// proxy.Dialer for it to make network requests.  FromURL may be passed to
// proxy.RegisterDialerType for the schemes "http" and "https".  The
// convenience function RegisterDialerFromURL simplifies this.
func FromURLBy(u *url.URL, forward proxy.Dialer) (proxy.Dialer, error) {
	return FromURLWithConfig(u, forward, nil)
}

// FromURLWithConfig is like New, but allows control over various options.
func FromURLWithConfig(u *url.URL, forward proxy.Dialer, cfg *Config) (proxy.Dialer, error) {
	var baseCfg BaseConfig
	if cfg == nil {
		cfg = DefaultConfig(u)
	}

	// set tls
	if cfg.Base.TLSClientConfig != nil {
		switch u.Scheme {
		case SchemeHTTPS, SchemeWebsocketSec:
			cfg.Base.TLSClientConfig = &tls.Config{
				ServerName:         u.Hostname(),
				InsecureSkipVerify: true,
			}
		default:
			// nothing
		}
	}

	baseDialer, err := baseDialerWithConfig(u, forward, baseCfg)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case SchemeWebsocket, SchemeWebsocketSec:
		return &websocket.Dialer{
			Dialer:        baseDialer,
			HeartInterval: cfg.WSHeartInterval,
		}, nil
	case SchemeHTTP, SchemeHTTPS:
		switch cfg.HTTPMethod {
		case http.MethodConnect:
			return &connect.Dialer{Dialer: baseDialer}, nil
		case http.MethodPost:
			return &post.Dialer{Dialer: baseDialer}, nil
		default:
			return nil, errors.Errorf("unsupported method for proxy: %s", cfg.HTTPMethod)
		}
	default:
		return nil, internal.ErrorUnsupportedScheme(errors.Errorf("unsupported scheme: %s", u.Scheme))
	}
}

func baseDialerWithConfig(u *url.URL, forward proxy.Dialer, cfg BaseConfig) (*base.Dialer, error) {
	// Make sure we have an allowable scheme.
	if supported, _ := supportedSchemes[u.Scheme]; !supported {
		err := errors.Errorf("unsupported scheme: %s" + u.Scheme)
		return nil, internal.ErrorUnsupportedScheme(err)
	}

	baseDialer := &base.Dialer{
		URL:             u,
		Forward:         forward,
		TLSClientConfig: cfg.TLSClientConfig,
		Header:          cfg.Header,
		DialTimeout:     cfg.DialTimeout,
	}

	if baseDialer.TLSClientConfig == nil {
		baseDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Work out the TLS server name
	if baseDialer.TLSClientConfig.ServerName == "" {
		h, _, err := net.SplitHostPort(u.Host)
		if nil != err && err.Error() == "missing port in address" {
			h = u.Host
		}
		baseDialer.TLSClientConfig.ServerName = h
	}

	// Parse out auth.
	if u.User != nil {
		baseDialer.HaveAuth = true
		baseDialer.Username = u.User.Username()
		baseDialer.Password, _ = u.User.Password()
	}

	// delete auth part in url for websocket scheme.
	if u.Scheme == SchemeWebsocket || u.Scheme == SchemeWebsocketSec {
		u.User = nil
	}

	return baseDialer, nil
}
