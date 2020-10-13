package tcpb

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	bufferLen    = 4196
)

// Bridge implemnt data flow tunnel between tcp client/server with websocket.
type Bridge struct {
	WSProxyGetter func(*http.Request) (*url.URL, error)
	HeartInterval time.Duration
}

// TCP2WS tcp client -> websocket tunnel
func (b *Bridge) TCP2WS(src net.Conn, wsURL string) error {
	wsDialer := &websocket.Dialer{
		Proxy:            b.WSProxyGetter,
		HandshakeTimeout: websocket.DefaultDialer.HandshakeTimeout,
	}

	wsCon, _, err := wsDialer.Dial(wsURL, nil)
	if err != nil {
		return errors.Wrapf(err, "dial ws %s failed", wsURL)
	}
	defer wsCon.Close()

	// send ping package timely.
	ticker := time.NewTicker(b.HeartInterval)
	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:				
				_ = wsCon.WriteMessage(websocket.PingMessage, nil)				
			}
		}
	}()

	pongWait := 2 * b.HeartInterval
	wsCon.SetPongHandler(func(string) error {
		return wsCon.SetReadDeadline(time.Now().Add(pongWait))
	})

	return syncCon(wsCon, src)
}

// WS2TCP websocket tunnel -> tcp server
func (b *Bridge) WS2TCP(src *websocket.Conn, tcpAddress string) error {
	tcpCon, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		return errors.Wrapf(err, "dial tcp %s failed", tcpAddress)
	}
	defer tcpCon.Close()

	return syncCon(src, tcpCon)
}

func syncCon(ws *websocket.Conn, tcp net.Conn) (err error) {
	errWS2tcp := ws2tcpWorker(ws, tcp)
	errTCP2ws := tcp2wsWorker(tcp, ws)

	select {
	case err = <-errWS2tcp:
		log.Printf("[INFO ] disconnected: ws://%s -> tcp://%s\n", ws.LocalAddr(), tcp.RemoteAddr())
	case err = <-errTCP2ws:
		log.Printf("[INFO ] disconnected: tcp://%s -> ws://%s\n", tcp.LocalAddr(), ws.RemoteAddr())
	}

	return err
}

func ctrlWorker(loopFn func() error) <-chan error {
	errCh := make(chan error)

	go func() {
		var err error
		for err == nil {
			err = loopFn()
		}

		errCh <- err
	}()

	return errCh
}

func ws2tcpWorker(from *websocket.Conn, to net.Conn) <-chan error {
	return ctrlWorker(func() error { return ws2tcp(from, to) })
}

func tcp2wsWorker(from net.Conn, to *websocket.Conn) <-chan error {
	return ctrlWorker(func() error { return tcp2ws(from, to) })
}

func ws2tcp(from *websocket.Conn, to net.Conn) error {
	_, buf, err := from.ReadMessage()
	if err != nil {
		return err
	}
	if len(buf) == 0 {
		return nil
	}

	wLen, err := to.Write(buf)
	if err != nil {
		return err
	}
	if wLen != len(buf) {
		return errors.Errorf("delivery byte length is not same: %d -> %d", len(buf), wLen)
	}

	return nil
}

func tcp2ws(from net.Conn, to *websocket.Conn) error {
	buf := make([]byte, bufferLen)
	n, err := from.Read(buf)	

	switch err {
	case nil, io.EOF:
		if n == 0 {
			return nil			
		}
		return to.WriteMessage(websocket.BinaryMessage, buf[:n])
	default:
		return err
	}
}
