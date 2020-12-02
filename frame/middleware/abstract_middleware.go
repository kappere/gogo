package middleware

import (
	"wataru.com/gogo/frame/context"
)

type Middleware interface {
	Before(c *context.Context)
	After(c *context.Context)
}
