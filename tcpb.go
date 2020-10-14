package tcpb

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

const (
	bufferLen = 4196
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

	return syncCon(wsCon, src, b.HeartInterval)
}

// WS2TCP websocket tunnel -> tcp server
func (b *Bridge) WS2TCP(src *websocket.Conn, tcpAddress string) error {
	tcpCon, err := net.Dial("tcp", tcpAddress)
	if err != nil {
		return errors.Wrapf(err, "dial tcp %s failed", tcpAddress)
	}
	defer tcpCon.Close()

	return syncCon(src, tcpCon, 0)
}

// wsHeartHandler send ping package timely to keep alive.
func wsHeartHandler(wsCon *websocket.Conn, interval time.Duration, wsWriteMux *sync.Mutex) chan<- bool {
	if interval <= 0 {
		return nil
	}

	pongWait := 2 * interval
	wsCon.SetPongHandler(func(string) error {
		return wsCon.SetReadDeadline(time.Now().Add(pongWait))
	})

	ticker := time.NewTicker(interval)
	stop := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				wsWriteMux.Lock()
				_ = wsCon.WriteMessage(websocket.PingMessage, nil)
				wsWriteMux.Unlock()
			case <-stop:
				ticker.Stop()
				break
			}
		}
	}()

	return stop
}

func syncCon(ws *websocket.Conn, tcp net.Conn, wsHeartInterval time.Duration) (err error) {
	wsWriteMutex := new(sync.Mutex)
	heartStop := wsHeartHandler(ws, wsHeartInterval, wsWriteMutex)

	tcp2wsLoopFn := func() error { return tcp2ws(tcp, ws) }
	if heartStop != nil {
		defer func() { heartStop <- true }()

		tcp2wsLoopFn = func() error {
			wsWriteMutex.Lock()
			defer wsWriteMutex.Unlock()
			return tcp2ws(tcp, ws)
		}
	}

	errWS2tcp := ctrlWorker(func() error { return ws2tcp(ws, tcp) })
	errTCP2ws := ctrlWorker(tcp2wsLoopFn)

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
