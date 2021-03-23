package post

import (
	"io"
	"net"
	"net/http/httputil"
)

// NewTrunkConn return a new.Conn implement for http chunked connection.
func NewTrunkConn(rawCon net.Conn) net.Conn {
	return trunkConn{
		rawCon,
		httputil.NewChunkedReader(rawCon),
		httputil.NewChunkedWriter(rawCon),
	}
}

// trunkConn wrapper http trunk data connection as net.Conn.
type trunkConn struct {
	net.Conn
	chunkedReader io.Reader
	chunkedWriter io.WriteCloser
}

func (ws trunkConn) Close() error {
	if err := ws.chunkedWriter.Close(); err != nil {
		defer ws.Conn.Close()
		return err
	}

	return ws.Conn.Close()
}

// Read implement net.Conn.
func (ws trunkConn) Read(b []byte) (n int, err error) {
	return ws.chunkedReader.Read(b)
}

// Write implement net.Conn.
func (ws trunkConn) Write(b []byte) (n int, err error) {
	return ws.chunkedWriter.Write(b)
}
