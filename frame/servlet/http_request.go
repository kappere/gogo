package servlet

import (
	"net/http"
)

type HttpRequest struct {
	*http.Request
}

func (httpRequest *HttpRequest) Uri() string {
	return httpRequest.URL.Path
}

func (httpRequest *HttpRequest) FullUri() string {
	return httpRequest.RequestURI
}

func NewHttpRequest(request *http.Request) *HttpRequest {
	return &HttpRequest{
		request,
	}
}
