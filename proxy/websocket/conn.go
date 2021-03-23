package websocket

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// NewWSConn return a new.Conn implement for *github.com/gorilla/websocket.Conn.
func NewWSConn(ws *websocket.Conn, wsHeartInterval time.Duration) net.Conn {
	if wsHeartInterval == 0 {
		return wsConn{ws, nil, nil}
	}

	writeMux := new(sync.Mutex)
	heartStop := wsHeartHandler(ws, wsHeartInterval, writeMux)

	log.Println("[DEBUG] ", "NewWSConn wrapped ok.")
	return wsConn{ws, writeMux, heartStop}
}

// wsConn wrap *github.com/gorilla/websocket.Conn with implement for net.Conn.
type wsConn struct {
	*websocket.Conn
	writeMux  *sync.Mutex
	heartStop chan<- bool
}

func (ws wsConn) Close() error {
	if ws.heartStop != nil {
		ws.heartStop <- true
	}

	return ws.Conn.Close()
}

// Read implement net.Conn.
func (ws wsConn) Read(b []byte) (n int, err error) {
	_, r, err := ws.NextReader()
	if err != nil {
		return n, err
	}

	return r.Read(b)
}

// Write implement net.Conn.
func (ws wsConn) Write(b []byte) (n int, err error) {
	if len(b) == 0 {
		return 0, nil
	}

	if ws.writeMux != nil {
		ws.writeMux.Lock()
		defer ws.writeMux.Unlock()
	}

	w, err := ws.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return 0, err
	}
	defer w.Close()

	return w.Write(b)
}

// SetDeadline implement net.Conn.
func (ws wsConn) SetDeadline(t time.Time) error {
	return ws.Conn.UnderlyingConn().SetDeadline(t)
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
		defer close(stop)

		for {
			select {
			case <-ticker.C:
				if wsWriteMux == nil {
					_ = wsCon.WriteMessage(websocket.PingMessage, nil)
				} else {
					wsWriteMux.Lock()
					_ = wsCon.WriteMessage(websocket.PingMessage, nil)
					wsWriteMux.Unlock()
				}
			case <-stop:
				log.Println("[INFO ] receive heart stop instruct")
				ticker.Stop()
				return
			}
		}
	}()

	return stop
}
