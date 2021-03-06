package middleware

import (
	"time"

	"wataru.com/gogo/frame/context"
	"wataru.com/gogo/logger"
)

type LogMiddleware struct {
}

// Before ...
func (middleware LogMiddleware) Before(c *context.Context) {
	c.LocalVars.Set("start_request_time", time.Now().UnixNano()/1000000)
	logger.Info("Process request [%s], client [%s]", c.HttpRequest.Uri(), c.ClientIP())
}

// After ...
func (middleware LogMiddleware) After(c *context.Context) {
	logger.Info("Process request [%s] complete, time %.0f ms",
		c.HttpRequest.Uri(),
		float64(time.Now().UnixNano()/1000000-c.LocalVars.Get("start_request_time").(int64)))
}

// NewLogMiddleware ...
func NewLogMiddleware() LogMiddleware {
	return LogMiddleware{}
}
