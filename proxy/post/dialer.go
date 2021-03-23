// Package post implement proxy.Dialer(s) with http post method.
package post

import (
	"net"
	"net/http"
	"net/http/httputil"

	"github.com/wuhuizuo/tcpb/proxy/base"
	"github.com/wuhuizuo/tcpb/proxy/internal"
)

// Dialer implement proxy.Dialer for http/https with post method.
type Dialer struct {
	*base.Dialer
}

// Dial connects to the given address via the server.
func (d *Dialer) Dial(network, _ string) (net.Conn, error) {
	nc, err := d.NewRawConn(network)
	if err != nil {
		return nil, err
	}

	req, err := d.NewHTTPRequest(http.MethodPost, "")
	if err != nil {
		_ = nc.Close()
		return nil, err
	}

	bs, err := httputil.DumpRequest(req, true)
	if err != nil {
		return nil, err
	}

	// write http request in raw tcp socket.
	_, err = nc.Write(bs)
	if err != nil {
		_ = nc.Close()
		return nil, err
	}

	// read response for connection initialization.
	if err := internal.AssertResponseFromRawConn(nc, req, d.DialTimeout, false); err != nil {
		_ = nc.Close()
		return nil, err
	}

	return NewTrunkConn(nc), nil
}
