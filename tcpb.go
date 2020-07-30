package tcpb

import (
	"io"
	"log"
	"net"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
)

// Bridge implemnt data flow tunnel between tcp client/server with websocket.
type Bridge struct {
	PackType int
}

// TCP2WS tcp client -> websocket tunnel
func (b *Bridge) TCP2WS(src net.Conn, wsURL string) error {
	defer src.Close()

	wsCon, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	wsCon.EnableWriteCompression(true)
	wsCon.UnderlyingConn()
	if err != nil {
		return errors.Wrapf(err, "dial ws %s failed", wsURL)
	}
	defer wsCon.Close()

	return Delive(src, wsCon)
}

// WS2TCP websocket tunnel -> tcp server
func (b *Bridge) WS2TCP(src *websocket.Conn, tcpAddress string) error {
	defer src.Close()

	tcpCon, err := net.Dial("tcp", tcpAddress)	
	if err != nil {
		return errors.Wrapf(err, "dial tcp %s failed", tcpAddress)
	}
	defer tcpCon.Close()

	return Delive(tcpCon, src)
}

// Delive tcp msg with tunnel
func Delive(con net.Conn, wsCon *websocket.Conn) error {
	// readWSType, wsReader, err := wsCon.NextReader()
	// if err != nil {
	// 	return errors.Wrap(err, "get websocket reader failed")
	// }
	// if readWSType != wsPackType {
	// 	return errors.Wrapf(err, "websocket message type is %d not eq %d", readWSType, wsPackType)
	// }

	// wsWriter, err := wsCon.NextWriter(wsPackType)
	// if err != nil {
	// 	return errors.Wrap(err, "get websocket writer failed")
	// }

	doneCh := make(chan bool)
	go copyWorker(con, wsCon.UnderlyingConn(), doneCh)
	go copyWorker(wsCon.UnderlyingConn(), con, doneCh)
	<-doneCh
	<-doneCh

	return nil
}

func copyWorker(dst io.Writer, src io.Reader, done chan<- bool) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Println("[ERROR] ", err)
	}

	done <- true
}
