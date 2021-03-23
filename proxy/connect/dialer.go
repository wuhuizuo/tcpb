// Package connect implement proxy.Dialer(s) with http connect method.
package connect

import (
	"net"
	"net/http"

	"github.com/wuhuizuo/tcpb/proxy/base"
	"github.com/wuhuizuo/tcpb/proxy/internal"
)

// Dialer implement proxy.Dialer for http/https with connect method.
type Dialer struct {
	*base.Dialer
}

// Dial connects to the given address via the server.
func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	nc, err := d.NewRawConn(network)
	if err != nil {
		return nil, err
	}

	req, err := d.NewHTTPRequest(http.MethodConnect, "Proxy-Authorization")
	if err != nil {
		_ = nc.Close()
		return nil, err
	}

	// send request.
	err = req.Write(nc)
	if err != nil {

		_ = nc.Close()
		return nil, err
	}

	// read response for connection initialization.
	if err := internal.AssertResponseFromRawConn(nc, req, d.DialTimeout, true); err != nil {
		_ = nc.Close()
		return nil, err
	}

	return nc, nil
}
