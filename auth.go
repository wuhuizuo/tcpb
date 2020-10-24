package tcpb

import "encoding/base64"

// HTTPUserInfo user info interface for access websocket using auth with http(s) protocol.
type HTTPUserInfo interface {
	HeaderKey() string
	HeaderValue() string
}

// HTTPBasicUserInfo user info for access websocket using basic auth.
type HTTPBasicUserInfo struct {
	Username string
	Password string
}

// HeaderKey implement HTTPUserInfo.
func (w *HTTPBasicUserInfo) HeaderKey() string {
	return "Authorization"
}

// HeaderValue implement HTTPUserInfo.
func (w *HTTPBasicUserInfo) HeaderValue() string {
	auth := w.Username + ":" + w.Password
	authEncode := base64.StdEncoding.EncodeToString([]byte(auth))
	return "Basic " + authEncode
}
