package internal

import (
	"bufio"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// AssertResponseFromRawConn assert http response in raw connection.
func AssertResponseFromRawConn(nc net.Conn, req *http.Request, timeout time.Duration, closeBody bool) error {
	conTimeouted := false
	connected := make(chan string)
	defer close(connected)

	// avoid read timeout.
	if timeout > 0 {
		go func() {
			select {
			case <-time.After(timeout):
				conTimeouted = true
			case <-connected:
			}
		}()
	}

	// Wait for a response.
	resp, err := http.ReadResponse(bufio.NewReader(nc), req)
	if err != nil {
		if conTimeouted {
			err = errors.Errorf("no connection to %q after %v", req.URL, timeout)
			return ErrorConnectionTimeout(err)
		}

		return err
	}
	if closeBody {
		defer resp.Body.Close()
	}

	// Make sure we can proceed.
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf(
			"non-OK status: %v",
			resp.Status,
		)
	}

	return nil
}
