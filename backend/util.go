package backend

import (
	"net/http"

	auth "github.com/abbot/go-http-auth"
)

func authReqToReq(authReq *auth.AuthenticatedRequest) *http.Request {
	return &http.Request{
		Method:           authReq.Method,
		URL:              authReq.URL,
		Proto:            authReq.Proto,
		ProtoMajor:       authReq.ProtoMajor,
		ProtoMinor:       authReq.ProtoMinor,
		Header:           authReq.Header,
		Body:             authReq.Body,
		GetBody:          authReq.GetBody,
		ContentLength:    authReq.ContentLength,
		TransferEncoding: authReq.TransferEncoding,
		Close:            authReq.Close,
		Host:             authReq.Host,
		Form:             authReq.Form,
		PostForm:         authReq.PostForm,
		MultipartForm:    authReq.MultipartForm,
		Trailer:          authReq.Trailer,
		RemoteAddr:       authReq.RemoteAddr,
		RequestURI:       authReq.RequestURI,
		TLS:              authReq.TLS,
		Cancel:           authReq.Cancel,
		Response:         authReq.Response,
	}
}

func reqToAuthReq(req *http.Request) *auth.AuthenticatedRequest {
	return &auth.AuthenticatedRequest{Request: *req}
}
