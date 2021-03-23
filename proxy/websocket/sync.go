package websocket

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// FIXME: there is a bug for websocket behind http proxy.

const (
	bufferLen = 32768 // 32 KByte
)

func SyncConn(ws *websocket.Conn, tcp net.Conn, wsHeartInterval time.Duration) (err error) {
	var wsWriteMutex *sync.Mutex

	if wsHeartInterval > 0 {
		wsWriteMutex = new(sync.Mutex)
		heartStop := wsHeartHandler(ws, wsHeartInterval, wsWriteMutex)
		defer func() { heartStop <- true }()
	}

	errWS2tcp := ctrlWorker(func() error { return ws2tcp(ws, tcp) })
	errTCP2ws := ctrlWorker(func() error { return tcp2ws(tcp, ws, wsWriteMutex) })

	select {
	case err = <-errWS2tcp:
		log.Printf("[INFO ] disconnected: ws://%s -> tcp://%s %+v\n", ws.LocalAddr(), tcp.RemoteAddr(), err)
	case err = <-errTCP2ws:
		log.Printf("[INFO ] disconnected: tcp://%s -> ws://%s %+v\n", tcp.LocalAddr(), ws.RemoteAddr(), err)
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

func tcp2ws(from net.Conn, to *websocket.Conn, wsWriteMux *sync.Mutex) error {
	buf := make([]byte, bufferLen)
	n, err := from.Read(buf)

	switch err {
	case nil, io.EOF:
		if n == 0 {
			return nil
		}

		if wsWriteMux == nil {
			return to.WriteMessage(websocket.BinaryMessage, buf[:n])
		}

		wsWriteMux.Lock()
		defer wsWriteMux.Unlock()
		return to.WriteMessage(websocket.BinaryMessage, buf[:n])
	default:
		log.Printf("[ERROR]] tcp2ws read err from tcp %s\n", err)
		return err
	}
}
