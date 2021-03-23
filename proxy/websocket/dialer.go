package websocket

import (
	"encoding/base64"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/wuhuizuo/tcpb/proxy/base"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Dialer implement proxy.Dialer for websocket.
type Dialer struct {
	*base.Dialer
	HeartInterval time.Duration
}

// Dial connects to the single proxy peer via the server.
func (d *Dialer) Dial(network, _ string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("only tcp supported")
	}

	wsDialer := &websocket.Dialer{
		HandshakeTimeout: d.DialTimeout,
		NetDial:          d.Forward.Dial,
		TLSClientConfig:  d.TLSClientConfig,
	}

	log.Println("[DEBUG] ", "websocket proxy url: ", d.URL.String())
	wsCon, _, err := wsDialer.Dial(d.URL.String(), d.newHeader())
	if err != nil {
		return nil, errors.Wrapf(err, "dial ws %s failed", d.URL)
	}

	log.Println("[DEBUG] ", "proxy dialed ok.")

	return NewWSConn(wsCon, d.HeartInterval), nil
}

func (d *Dialer) newHeader() http.Header {
	var ret http.Header

	if d.Header != nil {
		ret = d.Header.Clone()
	}
	if ret == nil {
		ret = make(http.Header)
	}

	for k, vv := range d.Header {
		for _, v := range vv {
			ret.Add(k, v)
		}
	}

	// set auth.
	if d.HaveAuth {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(d.Username+":"+d.Password))
		ret.Set("Authorization", basicAuth)
	}

	return ret
}
