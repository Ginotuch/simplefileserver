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

func reqToJson(req *http.Request) interface{} {
	return struct {
		URL        string
		Host       string
		RemoteAddr string
	}{URL: req.URL.String(), Host: req.Host, RemoteAddr: req.RemoteAddr}
}
