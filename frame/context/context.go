package context

import (
	"bytes"
	"container/list"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
	"wataru.com/gogo/config"
	"wataru.com/gogo/frame/servlet"
)

type Response struct {
	Data    interface{} `json:"data"`
	Code    int         `json:"code"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
}

type PageResponse struct {
	buffer *bytes.Buffer
}

func (r *PageResponse) GetBuffer() *bytes.Buffer {
	return r.buffer
}

type Context struct {
	HttpRequest  *servlet.HttpRequest
	HttpResponse *servlet.HttpResponse
	LocalVars    *LocalVars
	Params       Params
	// queryCache use url.ParseQuery cached the param query result from c.Request.URL.Query()
	queryCache url.Values

	// formCache use url.ParseQuery cached PostForm contains the parsed form data from POST, PATCH,
	// or PUT body parameters.
	formCache url.Values
}

type LocalVars struct {
	M map[string]interface{}
}

func (localVars *LocalVars) Set(key string, value interface{}) {
	localVars.M[key] = value
}

func (localVars *LocalVars) Get(key string) interface{} {
	return localVars.M[key]
}

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

// Get returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) Get(name string) (string, bool) {
	for _, entry := range ps {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return "", false
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) (va string) {
	va, _ = ps.Get(name)
	return
}

var (
	JSON = jsonBinding{}
	// XML           = xmlBinding{}
	// Form          = formBinding{}
	// Query         = queryBinding{}
	// FormPost      = formPostBinding{}
	// FormMultipart = formMultipartBinding{}
	// ProtoBuf      = protobufBinding{}
	// MsgPack       = msgpackBinding{}
	// YAML          = yamlBinding{}
	// Uri           = uriBinding{}
	// Header        = headerBinding{}
)

// Binding describes the interface which needs to be implemented for binding the
// data present in the request such as JSON request body, query parameters or
// the form POST.
type Binding interface {
	Name() string
	Bind(*http.Request, interface{}) error
}

// BindingBody adds BindBody method to Binding. BindBody is similar with Bind,
// but it reads the body from supplied bytes instead of req.Body.
type BindingBody interface {
	Binding
	BindBody([]byte, interface{}) error
}

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (jsonBinding) Bind(req *http.Request, obj interface{}) error {
	if req == nil || req.Body == nil {
		return fmt.Errorf("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func (jsonBinding) BindBody(body []byte, obj interface{}) error {
	return decodeJSON(bytes.NewReader(body), obj)
}

// EnableDecoderUseNumber is used to call the UseNumber method on the JSON
// Decoder instance. UseNumber causes the Decoder to unmarshal a number into an
// interface{} as a Number instead of as a float64.
var EnableDecoderUseNumber = false

// EnableDecoderDisallowUnknownFields is used to call the DisallowUnknownFields method
// on the JSON Decoder instance. DisallowUnknownFields causes the Decoder to
// return an error when the destination is a struct and the input contains object
// keys which do not match any non-ignored, exported fields in the destination.
var EnableDecoderDisallowUnknownFields = false

func decodeJSON(r io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(r)
	if EnableDecoderUseNumber {
		decoder.UseNumber()
	}
	if EnableDecoderDisallowUnknownFields {
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

type StructValidator interface {
	// ValidateStruct can receive any kind of type and it should never panic, even if the configuration is not right.
	// If the received type is not a struct, any validation should be skipped and nil must be returned.
	// If the received type is a struct or pointer to a struct, the validation should be performed.
	// If the struct is not valid or the validation itself fails, a descriptive error should be returned.
	// Otherwise nil must be returned.
	ValidateStruct(interface{}) error

	// Engine returns the underlying validator engine which powers the
	// StructValidator implementation.
	Engine() interface{}
}

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

var _ StructValidator = &defaultValidator{}

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	value := reflect.ValueOf(obj)
	valueType := value.Kind()
	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	if valueType == reflect.Struct {
		v.lazyinit()
		if err := v.validate.Struct(obj); err != nil {
			return err
		}
	}
	return nil
}

func (v *defaultValidator) Engine() interface{} {
	v.lazyinit()
	return v.validate
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

var Validator StructValidator = &defaultValidator{}

func validate(obj interface{}) error {
	if Validator == nil {
		return nil
	}
	return Validator.ValidateStruct(obj)
}

func (context *Context) Success(data interface{}) interface{} {
	return &Response{
		Data:    data,
		Code:    0,
		Success: true,
		Message: "",
	}
}

func (context *Context) Error(message string) interface{} {
	return &Response{
		Data:    nil,
		Code:    -1,
		Success: false,
		Message: message,
	}
}

func (context *Context) Render(templatePath string, data interface{}) interface{} {
	templateFile := config.ReadFile("templates/" + templatePath)
	tmpl, err := template.New("test").Parse(string(*templateFile))
	if err != nil {
		panic("Create template failed, err: " + err.Error())
	}
	resp := PageResponse{
		buffer: bytes.NewBuffer([]byte{}),
	}
	tmpl.Execute(resp.buffer, data)
	return &resp
}

/************************************/
/************ INPUT DATA ************/
/************************************/

// Param returns the value of the URL param.
// It is a shortcut for c.Params.ByName(key)
//     router.GET("/user/:id", func(c *gin.Context) {
//         // a GET request to /user/john
//         id := c.Param("id") // id == "john"
//     })
func (c *Context) Param(key string) string {
	return c.Params.ByName(key)
}

// Query returns the keyed url query value if it exists,
// otherwise it returns an empty string `("")`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//     GET /path?id=1234&name=Manu&value=
// 	   c.Query("id") == "1234"
// 	   c.Query("name") == "Manu"
// 	   c.Query("value") == ""
// 	   c.Query("wtf") == ""
func (c *Context) Query(key string) string {
	value, _ := c.GetQuery(key)
	return value
}

// DefaultQuery returns the keyed url query value if it exists,
// otherwise it returns the specified defaultValue string.
// See: Query() and GetQuery() for further information.
//     GET /?name=Manu&lastname=
//     c.DefaultQuery("name", "unknown") == "Manu"
//     c.DefaultQuery("id", "none") == "none"
//     c.DefaultQuery("lastname", "none") == ""
func (c *Context) DefaultQuery(key, defaultValue string) string {
	if value, ok := c.GetQuery(key); ok {
		return value
	}
	return defaultValue
}

// GetQuery is like Query(), it returns the keyed url query value
// if it exists `(value, true)` (even when the value is an empty string),
// otherwise it returns `("", false)`.
// It is shortcut for `c.Request.URL.Query().Get(key)`
//     GET /?name=Manu&lastname=
//     ("Manu", true) == c.GetQuery("name")
//     ("", false) == c.GetQuery("id")
//     ("", true) == c.GetQuery("lastname")
func (c *Context) GetQuery(key string) (string, bool) {
	if values, ok := c.GetQueryArray(key); ok {
		return values[0], ok
	}
	return "", false
}

// QueryArray returns a slice of strings for a given query key.
// The length of the slice depends on the number of params with the given key.
func (c *Context) QueryArray(key string) []string {
	values, _ := c.GetQueryArray(key)
	return values
}

func (c *Context) getQueryCache() {
	if c.queryCache == nil {
		c.queryCache = c.HttpRequest.Request.URL.Query()
	}
}

// GetQueryArray returns a slice of strings for a given query key, plus
// a boolean value whether at least one value exists for the given key.
func (c *Context) GetQueryArray(key string) ([]string, bool) {
	c.getQueryCache()
	if values, ok := c.queryCache[key]; ok && len(values) > 0 {
		return values, true
	}
	return []string{}, false
}

// QueryMap returns a map for a given query key.
func (c *Context) QueryMap(key string) map[string]string {
	dicts, _ := c.GetQueryMap(key)
	return dicts
}

// GetQueryMap returns a map for a given query key, plus a boolean value
// whether at least one value exists for the given key.
func (c *Context) GetQueryMap(key string) (map[string]string, bool) {
	c.getQueryCache()
	return c.get(c.queryCache, key)
}

// // PostForm returns the specified key from a POST urlencoded form or multipart form
// // when it exists, otherwise it returns an empty string `("")`.
// func (c *Context) PostForm(key string) string {
// 	value, _ := c.GetPostForm(key)
// 	return value
// }

// // DefaultPostForm returns the specified key from a POST urlencoded form or multipart form
// // when it exists, otherwise it returns the specified defaultValue string.
// // See: PostForm() and GetPostForm() for further information.
// func (c *Context) DefaultPostForm(key, defaultValue string) string {
// 	if value, ok := c.GetPostForm(key); ok {
// 		return value
// 	}
// 	return defaultValue
// }

// // GetPostForm is like PostForm(key). It returns the specified key from a POST urlencoded
// // form or multipart form when it exists `(value, true)` (even when the value is an empty string),
// // otherwise it returns ("", false).
// // For example, during a PATCH request to update the user's email:
// //     email=mail@example.com  -->  ("mail@example.com", true) := GetPostForm("email") // set email to "mail@example.com"
// // 	   email=                  -->  ("", true) := GetPostForm("email") // set email to ""
// //                             -->  ("", false) := GetPostForm("email") // do nothing with email
// func (c *Context) GetPostForm(key string) (string, bool) {
// 	if values, ok := c.GetPostFormArray(key); ok {
// 		return values[0], ok
// 	}
// 	return "", false
// }

// // PostFormArray returns a slice of strings for a given form key.
// // The length of the slice depends on the number of params with the given key.
// func (c *Context) PostFormArray(key string) []string {
// 	values, _ := c.GetPostFormArray(key)
// 	return values
// }

// func (c *Context) getFormCache() {
// 	if c.formCache == nil {
// 		c.formCache = make(url.Values)
// 		req := c.Request
// 		if err := req.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
// 			if err != http.ErrNotMultipart {
// 				debugPrint("error on parse multipart form array: %v", err)
// 			}
// 		}
// 		c.formCache = req.PostForm
// 	}
// }

// // GetPostFormArray returns a slice of strings for a given form key, plus
// // a boolean value whether at least one value exists for the given key.
// func (c *Context) GetPostFormArray(key string) ([]string, bool) {
// 	c.getFormCache()
// 	if values := c.formCache[key]; len(values) > 0 {
// 		return values, true
// 	}
// 	return []string{}, false
// }

// // PostFormMap returns a map for a given form key.
// func (c *Context) PostFormMap(key string) map[string]string {
// 	dicts, _ := c.GetPostFormMap(key)
// 	return dicts
// }

// // GetPostFormMap returns a map for a given form key, plus a boolean value
// // whether at least one value exists for the given key.
// func (c *Context) GetPostFormMap(key string) (map[string]string, bool) {
// 	c.getFormCache()
// 	return c.get(c.formCache, key)
// }

// get is an internal method and returns a map which satisfy conditions.
func (c *Context) get(m map[string][]string, key string) (map[string]string, bool) {
	dicts := make(map[string]string)
	exist := false
	for k, v := range m {
		if i := strings.IndexByte(k, '['); i >= 1 && k[0:i] == key {
			if j := strings.IndexByte(k[i+1:], ']'); j >= 1 {
				exist = true
				dicts[k[i+1:][:j]] = v[0]
			}
		}
	}
	return dicts, exist
}

// // FormFile returns the first file for the provided form key.
// func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
// 	if c.Request.MultipartForm == nil {
// 		if err := c.Request.ParseMultipartForm(c.engine.MaxMultipartMemory); err != nil {
// 			return nil, err
// 		}
// 	}
// 	f, fh, err := c.Request.FormFile(name)
// 	if err != nil {
// 		return nil, err
// 	}
// 	f.Close()
// 	return fh, err
// }

// // MultipartForm is the parsed multipart form, including file uploads.
// func (c *Context) MultipartForm() (*multipart.Form, error) {
// 	err := c.Request.ParseMultipartForm(c.engine.MaxMultipartMemory)
// 	return c.Request.MultipartForm, err
// }

// // SaveUploadedFile uploads the form file to specific dst.
// func (c *Context) SaveUploadedFile(file *multipart.FileHeader, dst string) error {
// 	src, err := file.Open()
// 	if err != nil {
// 		return err
// 	}
// 	defer src.Close()

// 	out, err := os.Create(dst)
// 	if err != nil {
// 		return err
// 	}
// 	defer out.Close()

// 	_, err = io.Copy(out, src)
// 	return err
// }

// // Bind checks the Content-Type to select a binding engine automatically,
// // Depending the "Content-Type" header different bindings are used:
// //     "application/json" --> JSON binding
// //     "application/xml"  --> XML binding
// // otherwise --> returns an error.
// // It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// // It decodes the json payload into the struct specified as a pointer.
// // It writes a 400 error and sets Content-Type header "text/plain" in the response if input is not valid.
// func (c *Context) Bind(obj interface{}) error {
// 	b := binding.Default(c.Request.Method, c.ContentType())
// 	return c.MustBindWith(obj, b)
// }

// BindJSON is a shortcut for c.MustBindWith(obj, binding.JSON).
func (c *Context) BindJSON(obj interface{}) error {
	return c.MustBindWith(obj, JSON)
}

// // BindXML is a shortcut for c.MustBindWith(obj, binding.BindXML).
// func (c *Context) BindXML(obj interface{}) error {
// 	return c.MustBindWith(obj, binding.XML)
// }

// // BindQuery is a shortcut for c.MustBindWith(obj, binding.Query).
// func (c *Context) BindQuery(obj interface{}) error {
// 	return c.MustBindWith(obj, binding.Query)
// }

// // BindYAML is a shortcut for c.MustBindWith(obj, binding.YAML).
// func (c *Context) BindYAML(obj interface{}) error {
// 	return c.MustBindWith(obj, binding.YAML)
// }

// // BindHeader is a shortcut for c.MustBindWith(obj, binding.Header).
// func (c *Context) BindHeader(obj interface{}) error {
// 	return c.MustBindWith(obj, binding.Header)
// }

// // BindUri binds the passed struct pointer using binding.Uri.
// // It will abort the request with HTTP 400 if any error occurs.
// func (c *Context) BindUri(obj interface{}) error {
// 	if err := c.ShouldBindUri(obj); err != nil {
// 		c.AbortWithError(http.StatusBadRequest, err).SetType(ErrorTypeBind) // nolint: errcheck
// 		return err
// 	}
// 	return nil
// }

// MustBindWith binds the passed struct pointer using the specified binding engine.
// It will abort the request with HTTP 400 if any error occurs.
// See the binding package.
func (c *Context) MustBindWith(obj interface{}, b Binding) error {
	if err := c.ShouldBindWith(obj, b); err != nil {
		panic(err)
	}
	return nil
}

// // ShouldBind checks the Content-Type to select a binding engine automatically,
// // Depending the "Content-Type" header different bindings are used:
// //     "application/json" --> JSON binding
// //     "application/xml"  --> XML binding
// // otherwise --> returns an error
// // It parses the request's body as JSON if Content-Type == "application/json" using JSON or XML as a JSON input.
// // It decodes the json payload into the struct specified as a pointer.
// // Like c.Bind() but this method does not set the response status code to 400 and abort if the json is not valid.
// func (c *Context) ShouldBind(obj interface{}) error {
// 	b := binding.Default(c.Request.Method, c.ContentType())
// 	return c.ShouldBindWith(obj, b)
// }

// // ShouldBindJSON is a shortcut for c.ShouldBindWith(obj, binding.JSON).
// func (c *Context) ShouldBindJSON(obj interface{}) error {
// 	return c.ShouldBindWith(obj, binding.JSON)
// }

// // ShouldBindXML is a shortcut for c.ShouldBindWith(obj, binding.XML).
// func (c *Context) ShouldBindXML(obj interface{}) error {
// 	return c.ShouldBindWith(obj, binding.XML)
// }

// // ShouldBindQuery is a shortcut for c.ShouldBindWith(obj, binding.Query).
// func (c *Context) ShouldBindQuery(obj interface{}) error {
// 	return c.ShouldBindWith(obj, binding.Query)
// }

// // ShouldBindYAML is a shortcut for c.ShouldBindWith(obj, binding.YAML).
// func (c *Context) ShouldBindYAML(obj interface{}) error {
// 	return c.ShouldBindWith(obj, binding.YAML)
// }

// // ShouldBindHeader is a shortcut for c.ShouldBindWith(obj, binding.Header).
// func (c *Context) ShouldBindHeader(obj interface{}) error {
// 	return c.ShouldBindWith(obj, binding.Header)
// }

// // ShouldBindUri binds the passed struct pointer using the specified binding engine.
// func (c *Context) ShouldBindUri(obj interface{}) error {
// 	m := make(map[string][]string)
// 	for _, v := range c.Params {
// 		m[v.Key] = []string{v.Value}
// 	}
// 	return binding.Uri.BindUri(m, obj)
// }

// ShouldBindWith binds the passed struct pointer using the specified binding engine.
// See the binding package.
func (c *Context) ShouldBindWith(obj interface{}, b Binding) error {
	return b.Bind(c.HttpRequest.Request, obj)
}

// // ShouldBindBodyWith is similar with ShouldBindWith, but it stores the request
// // body into the context, and reuse when it is called again.
// //
// // NOTE: This method reads the body before binding. So you should use
// // ShouldBindWith for better performance if you need to call only once.
// func (c *Context) ShouldBindBodyWith(obj interface{}, bb binding.BindingBody) (err error) {
// 	var body []byte
// 	if cb, ok := c.Get(BodyBytesKey); ok {
// 		if cbb, ok := cb.([]byte); ok {
// 			body = cbb
// 		}
// 	}
// 	if body == nil {
// 		body, err = ioutil.ReadAll(c.Request.Body)
// 		if err != nil {
// 			return err
// 		}
// 		c.Set(BodyBytesKey, body)
// 	}
// 	return bb.BindBody(body, obj)
// }

// // ClientIP implements a best effort algorithm to return the real client IP, it parses
// // X-Real-IP and X-Forwarded-For in order to work properly with reverse-proxies such us: nginx or haproxy.
// // Use X-Forwarded-For before X-Real-Ip as nginx uses X-Real-Ip with the proxy's IP.
// func (c *Context) ClientIP() string {
// 	if c.engine.ForwardedByClientIP {
// 		clientIP := c.requestHeader("X-Forwarded-For")
// 		clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
// 		if clientIP == "" {
// 			clientIP = strings.TrimSpace(c.requestHeader("X-Real-Ip"))
// 		}
// 		if clientIP != "" {
// 			return clientIP
// 		}
// 	}

// 	if c.engine.AppEngine {
// 		if addr := c.requestHeader("X-Appengine-Remote-Addr"); addr != "" {
// 			return addr
// 		}
// 	}

// 	if ip, _, err := net.SplitHostPort(strings.TrimSpace(c.Request.RemoteAddr)); err == nil {
// 		return ip
// 	}

// 	return ""
// }

// // ContentType returns the Content-Type header of the request.
// func (c *Context) ContentType() string {
// 	return filterFlags(c.requestHeader("Content-Type"))
// }

// // IsWebsocket returns true if the request headers indicate that a websocket
// // handshake is being initiated by the client.
// func (c *Context) IsWebsocket() bool {
// 	if strings.Contains(strings.ToLower(c.requestHeader("Connection")), "upgrade") &&
// 		strings.EqualFold(c.requestHeader("Upgrade"), "websocket") {
// 		return true
// 	}
// 	return false
// }

// func (c *Context) requestHeader(key string) string {
// 	return c.Request.Header.Get(key)
// }

type Initializer struct {
	F func()
}

var InitializerList = list.New()

func RegistInitializer(f func()) {
	InitializerList.PushBack(Initializer{
		F: f,
	})
}

func Inject(f func()) {
	InitializerList.PushBack(Initializer{
		F: f,
	})
}
