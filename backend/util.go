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
