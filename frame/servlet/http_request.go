package servlet

import "net/http"

type HttpRequest struct {
	Request *http.Request
}

func (httpRequest *HttpRequest) Uri() string {
	return httpRequest.Request.URL.Path
}

func (httpRequest *HttpRequest) FullUri() string {
	return httpRequest.Request.RequestURI
}

func NewHttpRequest(request *http.Request) *HttpRequest {
	return &HttpRequest{
		Request: request,
	}
}
