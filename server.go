package gogo

import (
	"wataru.com/gogo/frame/http"
)

var (
	httpServer *http.HttpServer = nil
)

// HttpServer : Run server.
func HttpServer() *http.HttpServer {
	if httpServer == nil {
		httpServer = new(http.HttpServer)
		httpServer.CreateRouter()
	}
	return httpServer
}
