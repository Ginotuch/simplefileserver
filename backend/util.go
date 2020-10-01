package backend

import (
	"net/http"

	auth "github.com/abbot/go-http-auth"
)

func authReqToReq(authReq *auth.AuthenticatedRequest) *http.Request {
	return &authReq.Request
}

func reqToAuthReq(req *http.Request) *auth.AuthenticatedRequest {
	return &auth.AuthenticatedRequest{Request: *req}
}

func reqToSafeStruct(req *http.Request) interface{} { // returns a struct that doesn't break zap
	return struct {
		URL        string
		Host       string
		RemoteAddr string
	}{URL: req.URL.String(), Host: req.Host, RemoteAddr: req.RemoteAddr}
}
