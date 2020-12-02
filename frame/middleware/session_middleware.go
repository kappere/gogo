package middleware

import (
	"net/http"

	"github.com/google/uuid"
	"wataru.com/gogo/config"
	"wataru.com/gogo/frame/context"
	"wataru.com/gogo/util"
)

type CookieConfig struct {
	name     string
	httpOnly bool
	maxAge   int
	// sameSite http.SameSite
	path   string
	secure bool
}

type SessionMiddleware struct {
	cookieConfig *CookieConfig
}

func (middleware SessionMiddleware) Before(c *context.Context) {
}

func (middleware SessionMiddleware) After(c *context.Context) {
	cookieConfig := middleware.cookieConfig
	if _, err := c.HttpRequest.Request.Cookie(cookieConfig.name); err != nil {
		cookie := http.Cookie{
			Name:     cookieConfig.name,
			Value:    uuid.New().String(),
			Path:     cookieConfig.path,
			MaxAge:   cookieConfig.maxAge,
			Secure:   cookieConfig.secure,
			HttpOnly: cookieConfig.httpOnly,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(c.HttpResponse.ResponseWriter(), &cookie)
	}
}

func NewSessionMiddleware() SessionMiddleware {
	serverConf := util.ValueOrDefault((*config.GlobalConfig.Map)["server"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	cookieConf := util.ValueOrDefault(serverConf["cookie"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	return SessionMiddleware{
		cookieConfig: &CookieConfig{
			name:     util.ValueOrDefault(cookieConf["name"], "SESSION").(string),
			httpOnly: util.ValueOrDefault(cookieConf["http-only"], false).(bool),
			maxAge:   util.ValueOrDefault(cookieConf["max-age"], 2147483647).(int),
			path:     util.ValueOrDefault(cookieConf["path"], "/").(string),
			secure:   util.ValueOrDefault(cookieConf["secure"], false).(bool),
		},
	}
}
