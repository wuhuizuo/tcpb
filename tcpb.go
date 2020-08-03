package tcpb

import (
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Bridge implemnt data flow tunnel between tcp client/server with websocket.
type Bridge struct {
	WSProxyGetter func(*http.Request) (*url.URL, error)
}

// TCP2WS tcp client -> websocket tunnel
func (b *Bridge) TCP2WS(src net.Conn, wsURL string) error {
	defer src.Close()

	wsDialer := &websocket.Dialer{
		Proxy:            b.WSProxyGetter,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}

	wsCon, _, err := wsDialer.Dial(wsURL, nil)
	if err != nil {
		return errors.Wrapf(err, "dial ws %s failed", wsURL)
	}

	defer wsCon.Close()

	return Delivery(src, wsCon)
}

// WS2TCP websocket tunnel -> tcp server
func (b *Bridge) WS2TCP(src *websocket.Conn, tcpAddress string) error {
	defer src.Close()

	tcpCon, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		return errors.Wrapf(err, "dial tcp %s failed", tcpAddress)
	}
	defer tcpCon.Close()

	return Delivery(tcpCon, src)
}

// Delivery tcp msg with tunnel
func Delivery(con net.Conn, wsCon *websocket.Conn) error {
	return syncCon(con, wsCon.UnderlyingConn())
}

func syncCon(conA, conB net.Conn) error {
	conLen := 2
	doneCh := make(chan bool, conLen)
	errCh := make(chan error, conLen)

	go copyWorker(conA, conB, doneCh, errCh)
	go copyWorker(conB, conA, doneCh, errCh)

	var err error
	for i := 0; i < conLen; i++ {
		<-doneCh

		if err == nil {
			err = <-errCh
		} else {
			<-errCh
		}
	}

	return err
}

func copyWorker(dst io.Writer, src io.Reader, done chan<- bool, err chan<- error) {
	_, ioErr := io.Copy(dst, src)
	done <- true

	if ioErr != nil {
		ioErr = errors.Wrap(ioErr, "io copy error between connections")
	}
	err <- ioErr
}
