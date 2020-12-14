package middleware

import (
	"net/http"

	"wataru.com/gogo/config"
	"wataru.com/gogo/frame/context"
	"wataru.com/gogo/frame/session"
	"wataru.com/gogo/util"
)

type CookieConfig struct {
	name     string
	httpOnly bool
	maxAge   int
	sameSite http.SameSite
	path     string
	secure   bool
}

type SessionMiddleware struct {
	cookieConfig *CookieConfig
}

var sessionMap map[string]*session.Session = make(map[string]*session.Session)

func (middleware SessionMiddleware) Before(c *context.Context) {
	cookieConfig := middleware.cookieConfig
	sessionIDValue, err := c.HttpRequest.Cookie(cookieConfig.name)
	var ss *session.Session
	if err != nil {
		ss = createNewSession()
	} else {
		if ss = sessionMap[sessionIDValue.Value]; ss == nil {
			ss = createNewSession()
		}
	}
	c.Session = ss
}

func (middleware SessionMiddleware) After(c *context.Context) {
	cookieConfig := middleware.cookieConfig
	if c.Session.IsNew {
		cookie := http.Cookie{
			Name:     cookieConfig.name,
			Path:     cookieConfig.path,
			MaxAge:   cookieConfig.maxAge,
			Secure:   cookieConfig.secure,
			HttpOnly: cookieConfig.httpOnly,
			SameSite: cookieConfig.sameSite,
			Value:    c.Session.Id,
		}
		http.SetCookie(c.HttpResponse.ResponseWriter(), &cookie)
		c.Session.IsNew = false
	}
}

func createNewSession() *session.Session {
	ss := session.CreateNewSession()
	sessionMap[ss.Id] = ss
	return ss
}

func NewSessionMiddleware() SessionMiddleware {
	serverConf := util.ValueOrDefault((*config.GlobalConfig.Map)["server"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	cookieConf := util.ValueOrDefault(serverConf["cookie"], make(map[interface{}]interface{})).(map[interface{}]interface{})
	return SessionMiddleware{
		cookieConfig: &CookieConfig{
			name:     util.ValueOrDefault(cookieConf["name"], "SESSIONID").(string),
			httpOnly: util.ValueOrDefault(cookieConf["http-only"], true).(bool),
			maxAge:   util.ValueOrDefault(cookieConf["max-age"], 2147483647).(int),
			path:     util.ValueOrDefault(cookieConf["path"], "/").(string),
			secure:   util.ValueOrDefault(cookieConf["secure"], false).(bool),
			sameSite: http.SameSiteLaxMode,
		},
	}
}
