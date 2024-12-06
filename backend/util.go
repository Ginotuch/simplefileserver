package backend

import (
	"net/http"
)

func reqToSafeStruct(req *http.Request) interface{} {
	return struct {
		URL        string
		Host       string
		RemoteAddr string
	}{URL: req.URL.String(), Host: req.Host, RemoteAddr: req.RemoteAddr}
}
