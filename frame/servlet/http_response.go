package servlet

import "net/http"

type HttpResponse struct {
	responseWriter http.ResponseWriter
}

func (httpResponse *HttpResponse) ResponseWriter() http.ResponseWriter {
	return httpResponse.responseWriter
}

func NewHttpResponse(responseWriter http.ResponseWriter) *HttpResponse {
	return &HttpResponse{
		responseWriter: responseWriter,
	}
}
