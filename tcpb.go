package tcpb

import (
	"encoding/base64"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	ws "github.com/wuhuizuo/tcpb/proxy/websocket"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	netproxy "golang.org/x/net/proxy"
)

const (
	bufferLen = 32768 // 32 KByte
)

// Bridge implement data flow tunnel between tcp client/server with websocket.
type Bridge struct {
	WSProxyGetter func(*http.Request) (*url.URL, error)
	HeartInterval time.Duration
}

// TCP2Proxy tcp client -> tcp proxy tunnel(http/https/http2.0 or socket5).
func (b *Bridge) TCP2Proxy(src net.Conn, proxyURL string) error {
	if strings.HasPrefix(proxyURL, "ws://") || strings.HasPrefix(proxyURL, "wss://") {
		return b.TCP2WS(src, proxyURL)
	}

	dialProxyURL, err := url.Parse(proxyURL)
	if err != nil {
		return err
	}

	proxyDialer, err := netproxy.FromURL(dialProxyURL, netproxy.FromEnvironment())
	if err != nil {
		return err
	}

	remoteCon, err := proxyDialer.Dial("tcp", "")
	if err != nil {
		return err
	}
	defer remoteCon.Close()

	return syncConn(remoteCon, src)
}

// WS2TCP websocket tunnel -> tcp server
func (b *Bridge) WS2TCP(src *websocket.Conn, tcpAddress string) error {
	tcpCon, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		return errors.Wrapf(err, "dial tcp %s failed", tcpAddress)
	}
	defer func() {
		err := tcpCon.Close()
		if err != nil {
			log.Printf("[WARN ] connection : ws://%s -> tcp://%s close error %+v\n", src.LocalAddr(), tcpCon.RemoteAddr(), err)
		}
	}()

	return syncConn(tcpCon, ws.NewWSConn(src, b.HeartInterval))
}

func syncConn(a, b net.Conn) (err error) {
	errCh1 := make(chan error)
	errCh2 := make(chan error)

	go func() {
		_, err := io.Copy(a, b)
		errCh1 <- err
	}()
	go func() {
		_, err := io.Copy(b, a)
		errCh2 <- err
	}()

	select {
	case err = <-errCh1:
		log.Printf("[WARN ] disconnected: tcp://%s -> tcp://%s %+v\n", a.LocalAddr(), b.RemoteAddr(), err)
	case err = <-errCh2:
		log.Printf("[WARN ] disconnected: tcp://%s -> tcp://%s %+v\n", b.LocalAddr(), a.RemoteAddr(), err)
	}

	return err
}

// TCP2WS tcp client -> websocket tunnel
func (b *Bridge) TCP2WS(src net.Conn, wsURL string) error {
	wsDialer := &websocket.Dialer{
		Proxy:            b.WSProxyGetter,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}

	u, err := url.Parse(wsURL)
	if err != nil {
		return errors.WithStack(err)
	}

	// set auth.
	var wsHeader http.Header
	if user := u.User; user != nil {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(user.String()))
		wsHeader = make(http.Header)
		wsHeader.Set("Authorization", basicAuth)
	}

	wsCon, _, err := wsDialer.Dial(wsURL, wsHeader)
	if err != nil {
		return errors.Wrapf(err, "dial ws %s failed", wsURL)
	}
	defer wsCon.Close()

	return ws.SyncConn(wsCon, src, b.HeartInterval)
}
