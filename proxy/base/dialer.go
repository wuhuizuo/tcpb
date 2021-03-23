package base

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/wuhuizuo/tcpb/proxy/internal"
	"golang.org/x/net/proxy"
)

// Config dialer config.
type Config struct {
}

// Dialer implement basic dialer for http protocol.
type Dialer struct {
	URL             *url.URL
	Forward         proxy.Dialer
	TLSClientConfig *tls.Config

	// Header sets the headers in the initial HTTP CONNECT request.  See
	// the documentation for http.Request for more information.
	Header http.Header

	// DialTimeout is an optional timeout for connections through (not to)
	// the proxy server.
	DialTimeout time.Duration

	HaveAuth bool
	Username string
	Password string
}

// NewRawConn create a raw connection with http(s) proxy.
func (d *Dialer) NewRawConn(network string) (net.Conn, error) {
	proxyAddr := d.URL.Host
	if d.URL.Port() == "" {
		if d.URL.Scheme == "http" {
			proxyAddr += ":80"
		}
		if d.URL.Scheme == "https" {
			proxyAddr += ":443"
		}
	}

	nc, err := d.Forward.Dial(network, proxyAddr)
	if nil != err {
		return nil, err
	}

	/// Upgrade to TLS if necessary.
	if d.URL.Scheme == "https" {
		nc = tls.Client(nc, d.TLSClientConfig)
	}

	return nc, nil
}

func (d *Dialer) FillHeaderToReq(req *http.Request, authHeadKey string) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}

	for k, vv := range d.Header {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// set auth.
	if d.HaveAuth {
		internal.SetBasicAuth(req, d.Username, d.Password, authHeadKey)
	}
}

// NewHTTPRequest return a *http.Request compose from remote address and other args.
func (d *Dialer) NewHTTPRequest(httpMethod, authHeadKey string) (*http.Request, error) {
	req := &http.Request{
		Method:     httpMethod,
		URL:        d.URL,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       d.URL.Host,
	}

	req.TransferEncoding = append(req.TransferEncoding, "chunked")
	d.FillHeaderToReq(req, authHeadKey)

	return req, nil
}

// SchemeDialerGenerator is Dialer generator for `proxy.RegisterDialerType`.
type SchemeDialerGenerator func(*url.URL, proxy.Dialer) (proxy.Dialer, error)
