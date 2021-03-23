package internal

import (
	"encoding/base64"
	"net/http"
)

// SetBasicAuth set auth info for http request.
// 	customAuthKey default is "Authorization",
//	you can set with custom key, for example: "Proxy-Authorization".
func SetBasicAuth(req *http.Request, user, passwd, customAuthKey string) {
	if req == nil || user == "" || passwd == "" {
		return
	}

	if customAuthKey == "" {
		req.SetBasicAuth(user, passwd)
	} else {
		basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+passwd))
		req.Header.Add(customAuthKey, basicAuth)
	}
}
