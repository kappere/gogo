package router

import (
	"container/list"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"wataru.com/gogo/frame/context"
	"wataru.com/gogo/frame/middleware"
	"wataru.com/gogo/frame/panics"
	"wataru.com/gogo/frame/servlet"
	"wataru.com/gogo/json"
	"wataru.com/gogo/logger"
)

type HttpMethodType string

const (
	ALL    HttpMethodType = "ALL"
	GET    HttpMethodType = "GET"
	POST   HttpMethodType = "POST"
	PUT    HttpMethodType = "PUT"
	DELETE HttpMethodType = "DELETE"
)

type HandlerFunc struct {
	target         reflect.Value
	targetName     string
	httpMethodType HttpMethodType
	middlewares    *list.List // 细粒度全局中间件
	groups         *list.List // 路由组
}

type Router struct {
	handlers    map[string]http.Handler
	handleFuncs map[string]*HandlerFunc
	middlewares *list.List // 全局中间件
}

type RouterGroup struct {
	path        string
	router      *Router
	middlewares *list.List // 分组中间件
	accessors   *list.List
}

func (router *Router) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	urlPath := req.URL.Path
	if hl, ok := router.handlers[urlPath]; ok {
		hl.ServeHTTP(resp, req)
		return
	}
	if fn, ok := router.handleFuncs[urlPath]; ok {
		if ALL != fn.httpMethodType && req.Method != string(fn.httpMethodType) {
			http.Error(resp, "405 method '"+req.Method+"' not allowed", http.StatusMethodNotAllowed)
			return
		}
		router.serve(resp, req, fn)
		return
	}
	http.NotFound(resp, req)
}

func (router *Router) InitRouterMiddleware() {
	router.loadDefaultMiddleware()
	for _, fcs := range router.handleFuncs {
		middlewares := router.collectMiddleware(fcs.groups)
		fcs.middlewares = middlewares
	}
}

func (router *Router) loadDefaultMiddleware() {
	router.Middleware(middleware.NewLogMiddleware())
	router.Middleware(middleware.NewSessionMiddleware())
}

func (router *Router) collectMiddleware(groups *list.List) *list.List {
	middlewares := list.New()
	for i := router.middlewares.Front(); i != nil; i = i.Next() {
		middlewares.PushBack(i.Value)
	}
	if groups != nil {
		for i := groups.Front(); i != nil; i = i.Next() {
			for j := i.Value.(*RouterGroup).middlewares.Front(); j != nil; j = j.Next() {
				middlewares.PushBack(j.Value)
			}
		}
	}
	return middlewares
}

func (router *Router) serve(resp http.ResponseWriter, req *http.Request, handlerFunc *HandlerFunc) {
	middlewares := handlerFunc.middlewares
	httpRequest := servlet.NewHttpRequest(req)
	httpResponse := servlet.NewHttpResponse(resp)
	c := &context.Context{
		HttpRequest:  httpRequest,
		HttpResponse: httpResponse,
		LocalVars: &context.LocalVars{
			M: make(map[string]interface{}),
		},
	}
	for i := middlewares.Front(); i != nil; i = i.Next() {
		i.Value.(middleware.Middleware).Before(c)
	}
	r, err := router.invokeTargetControllerMethod(c, handlerFunc)
	if err != nil {
		r = c.Error(err.Error())
	}
	for i := middlewares.Front(); i != nil; i = i.Next() {
		i.Value.(middleware.Middleware).After(c)
	}
	resp.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp.WriteHeader(http.StatusOK)
	resp.Write([]byte(json.ToJsonByte(r)))
}

func getFunctionName(i interface{}, seps ...rune) string {
	fn := runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
	fields := strings.FieldsFunc(fn, func(sep rune) bool {
		for _, s := range seps {
			if sep == s {
				return true
			}
		}
		return false
	})
	if size := len(fields); size > 0 {
		w := fields[size-1]
		return w[0 : len(w)-3]
	}
	return ""
}

func (router *Router) invokeTargetControllerMethod(c *context.Context, handlerFunc *HandlerFunc) (result interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			var msg string
			_, e := r.(*panics.BizPanic)
			if e {
				msg = fmt.Sprintf("%s", r)
			} else {
				msg = "服务器异常，请联系管理员"
			}
			logger.Error("Web exception for error: %s", msg)
			err = errors.New(msg)
		}
	}()
	method := handlerFunc.target
	args := []reflect.Value{reflect.ValueOf(c)}
	result = method.Call(args)[0].Interface()
	return result, nil
}

func (router *Router) Handle(pattern string, hl http.Handler) {
	router.handlers[pattern] = hl
}

func (router *Router) HandleFunc(
	httpMethodType HttpMethodType,
	pattern string,
	controller interface{},
	fn func(c *context.Context) *context.Response,
	groups *list.List) {
	fnName := getFunctionName(fn, '/', '.')
	method := reflect.ValueOf(controller).MethodByName(fnName)
	targetName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
	router.handleFuncs[pattern] = &HandlerFunc{
		target:         method,
		targetName:     targetName,
		httpMethodType: httpMethodType,
		middlewares:    nil,
		groups:         groups,
	}
}

func (router *Router) LogRouterSummary() {
	for k, v := range router.handleFuncs {
		middlewareNames := make([]string, v.middlewares.Len())
		t := 0
		for i := v.middlewares.Front(); i != nil; i = i.Next() {
			mw := reflect.TypeOf(i.Value)
			middlewareNames[t] = mw.Name()
			t++
		}
		logger.Raw("%8sMapping [%3s] [%-20s] => [%-40s] middlewares:%s", "", v.httpMethodType, k, v.targetName, middlewareNames)
	}
}

func (router *Router) Middleware(mw middleware.Middleware) {
	router.middlewares.PushBack(mw)
}

func (router *Router) Method(httpMethodType HttpMethodType, path string, controller interface{}, controllerFunc func(c *context.Context) *context.Response) {
	// Handle("/hello/golang/", &BaseHander{})
	router.HandleFunc(httpMethodType, path, controller, controllerFunc, nil)
}

// All 不限制HTTP方法路由注册
func (router *Router) All(path string, controller *interface{}, controllerFunc func(c *context.Context) *context.Response) {
	router.Method(ALL, path, controller, controllerFunc)
}

// Get HTTPGET路由注册
func (router *Router) Get(path string, controller *interface{}, controllerFunc func(c *context.Context) *context.Response) {
	router.Method(GET, path, controller, controllerFunc)
}

// Post HTTPGET路由注册
func (router *Router) Post(path string, controller *interface{}, controllerFunc func(c *context.Context) *context.Response) {
	router.Method(POST, path, controller, controllerFunc)
}

// Group 分组路由注册
func (router *Router) Group(path string, groupFunc func(group *RouterGroup)) {
	accessors := list.New()
	newGroup := &RouterGroup{
		path:        path,
		router:      router,
		middlewares: list.New(),
		accessors:   accessors,
	}
	accessors.PushBack(newGroup)
	groupFunc(newGroup)
}

// Group 分组路由注册
func (group *RouterGroup) Group(path string, groupFunc func(group *RouterGroup)) {
	accessors := list.New()
	for i := group.accessors.Front(); i != nil; i = i.Next() {
		accessors.PushBack(i.Value)
	}
	newGroup := &RouterGroup{
		path:        path,
		router:      group.router,
		middlewares: list.New(),
		accessors:   accessors,
	}
	accessors.PushBack(newGroup)
	groupFunc(newGroup)
}

func (group *RouterGroup) Middleware(mw middleware.Middleware) {
	group.middlewares.PushBack(mw)
}

func (group *RouterGroup) Method(httpMethodType HttpMethodType, path string, controller interface{}, controllerFunc func(c *context.Context) *context.Response) {
	group.router.HandleFunc(httpMethodType, concatRouterPath(group.path, path), controller, controllerFunc, group.accessors)
}

func (group *RouterGroup) All(path string, controller interface{}, controllerFunc func(c *context.Context) *context.Response) {
	group.Method(ALL, path, controller, controllerFunc)
}

func (group *RouterGroup) Get(path string, controller interface{}, controllerFunc func(c *context.Context) *context.Response) {
	group.Method(GET, path, controller, controllerFunc)
}

func (group *RouterGroup) Post(path string, controller interface{}, controllerFunc func(c *context.Context) *context.Response) {
	group.Method(POST, path, controller, controllerFunc)
}

func NewRouter() *Router {
	return &Router{
		handlers:    make(map[string]http.Handler),
		handleFuncs: make(map[string]*HandlerFunc),
		middlewares: list.New(),
	}
}

func concatRouterPath(p1, p2 string) string {
	if len(p1) > 0 && len(p2) > 0 && p1[len(p1)-1] == '/' && p2[0] == '/' {
		return p1 + p2[1:]
	}
	return p1 + p2
}
