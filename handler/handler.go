package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"time"
)

var (
	// Create-Handler-Implementation-Function table.
	handlerFunc = make(map[string]NewHandlerFunc)
	// DefaultForwarder register name.
	defaultForwarderName = HandlerName(&DefaultForwarder{})
	// DefaultInterceptor register name.
	defaultInterceptorRegisterName = HandlerName(&DefaultInterceptor{})
	// DefaultNotFound register name.
	defaultNotFoundRegisterName = HandlerName(&DefaultNotFound{})
)

func init() {
	// Register Create-Handler-Implementation-Function
	RegisterHandler(defaultForwarderName, NewDefaultForwarder)
	RegisterHandler(defaultInterceptorRegisterName, NewDefaultInterceptor)
	RegisterHandler(defaultNotFoundRegisterName, NewDefaultNotFound)
}

// The data passed in Handler call chain.
type Context struct {
	Res http.ResponseWriter
	Req *http.Request
	// Service route path.
	// Top dir of request url path.
	Path string
	// Used for save and pass temp data in Handler call chain.
	Data interface{}
}

type Handler interface {
	// Return false will abort call chain.
	Handle(*Context) bool
	// Update self data.
	Update(interface{}) error
	// Release resource
	Release()
}

// Create-Handler-Implementation function.
// Arg data is initial data.
type NewHandlerFunc func(data interface{}) (Handler, error)

// Register NewHandlerFunc function,name is key.
func RegisterHandler(name string, newFunc NewHandlerFunc) {
	handlerFunc[name] = newFunc
}

// Create Handler by name.If name is empty string,create DefaultForwarder.
// Arg data will pass to NewHandlerFunc
func NewHandler(name string, data interface{}) (Handler, error) {
	newFunc, ok := handlerFunc[name]
	if !ok {
		return NewDefaultForwarder(data)
	}
	h, err := newFunc(data)
	if err != nil {
		return nil, fmt.Errorf(`"%s" %s`, name, err.Error())
	}
	return h, nil
}

// Return Handler h package path and struct name.Use for RegisterHandler function.
// Example: DefaultForwarder return "github.com/qq51529210/gateway/handler.DefaultForwarder".
func HandlerName(h Handler) string {
	_type := reflect.TypeOf(h).Elem()
	return _type.PkgPath() + "." + _type.Name()
}

// Return DefaultForwarder register name.
func DefaultForwarderName() string {
	return defaultForwarderName
}

// DefaultForwarder initial data.
type NewDefaultForwarderData struct {
	// Http service url like "http://host/src?a=1".
	// Must be define.
	RequestUrl string `json:"requestUrl"`
	// Forward request timeout,millisecond.
	RequestTimeout int `json:"requestTimeout"`
	// Which heads will be forward.
	// If it's empty,forward all headers.
	RequestHeader []string `json:"requestHeader"`
	// Addition heads add to forward request.
	RequestAdditionHeader map[string]string `json:"requestAdditionHeader"`
	// Addition heads add to forward response.
	ResponseAdditionHeader map[string]string `json:"responseAdditionHeader"`
}

// Forward HTTP request.
type DefaultForwarder struct {
	// Foward url.
	RequestUrl *url.URL
	// Forward request timeout,millisecond.
	RequestTimeout time.Duration
	// Which heads will be forward.
	// If it's empty,forward all headers.
	RequestHeader map[string]int
	// Addition heads add to forward request.
	RequestAdditionHeader map[string]string
	// Addition heads add to forward response.
	ResponseAdditionHeader map[string]string
}

func (h *DefaultForwarder) Handle(c *Context) bool {
	// Init forward http.Request.
	var request http.Request
	request.Method = c.Req.Method
	request.URL = new(url.URL)
	*request.URL = *h.RequestUrl
	request.URL.Path = c.Req.URL.Path[len(c.Path):]
	request.URL.RawQuery = c.Req.URL.RawQuery
	request.URL.RawFragment = c.Req.URL.RawFragment
	request.ContentLength = c.Req.ContentLength
	// Forward headers.
	request.Header = make(http.Header)
	if len(h.RequestHeader) < 1 {
		// Forward all headers.
		for k := range c.Req.Header {
			request.Header.Set(k, c.Req.Header.Get(k))
		}
	} else {
		// Forward specified headers.
		for k := range h.RequestHeader {
			v := c.Req.Header.Get(k)
			if v != "" {
				request.Header.Set(k, v)
			}
		}
	}
	// Addition headers
	for k, v := range h.RequestAdditionHeader {
		request.Header.Set(k, v)
	}
	request.Body = c.Req.Body
	// Do request.
	client := &http.Client{Timeout: h.RequestTimeout}
	response, err := client.Do(&request)
	if err != nil {
		fmt.Println(err)
		return false
	}
	// Response headers.
	header := c.Res.Header()
	for k, v := range response.Header {
		for _, s := range v {
			header.Add(k, s)
		}
	}
	// Response addition headers.
	for k, v := range h.ResponseAdditionHeader {
		header.Add(k, v)
	}
	c.Res.WriteHeader(response.StatusCode)
	io.Copy(c.Res, response.Body)
	return true
}

// Arg data is *DefaultForwarderData type.
func (h *DefaultForwarder) Update(data interface{}) error {
	if data == nil {
		return errors.New(`data must be defined`)
	}
	d, ok := data.(*NewDefaultForwarderData)
	if !ok {
		return errors.New(`data must be "*DefaultForwarderData" type`)
	}
	// requestUrl
	requestUrl, err := url.Parse(d.RequestUrl)
	if err != nil {
		return fmt.Errorf(`"requestUrl" %s`, err.Error())
	}
	h.RequestUrl = requestUrl
	// requestTimeout
	if d.RequestTimeout >= 0 {
		h.RequestTimeout = time.Duration(d.RequestTimeout) * time.Millisecond
	}
	// requestHeader
	if len(d.RequestHeader) > 0 {
		h.RequestHeader = make(map[string]int)
		for _, s := range d.RequestHeader {
			h.RequestHeader[s] = 1
		}
	}
	// requestAdditionHeader
	if len(d.RequestAdditionHeader) > 0 {
		h.RequestAdditionHeader = make(map[string]string)
		for k, v := range d.RequestAdditionHeader {
			h.RequestAdditionHeader[k] = v
		}
	}
	// responseAdditionHeader
	if len(d.ResponseAdditionHeader) > 0 {
		h.ResponseAdditionHeader = make(map[string]string)
		for k, v := range d.ResponseAdditionHeader {
			h.ResponseAdditionHeader[k] = v
		}
	}
	return nil
}

func (h *DefaultForwarder) Release() {}

// Create DefaultForwarder function.
func NewDefaultForwarder(data interface{}) (Handler, error) {
	var d *NewDefaultForwarderData
	switch v := data.(type) {
	case *NewDefaultForwarderData:
		d = v
	case map[string]interface{}:
		d = new(NewDefaultForwarderData)
		err := Map2Struct(v, d)
		if err != nil {
			return nil, err
		}
	case string:
		// Json string.
		err := json.Unmarshal([]byte(v), d)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupport data type %s", reflect.TypeOf(data).Kind().String())
	}

	h := new(DefaultForwarder)
	err := h.Update(d)
	if err != nil {
		return nil, err
	}
	if h.RequestUrl == nil {
		return nil, errors.New(`"requestUrl" must be defined`)
	}
	return h, nil
}

// Return DefaultInterceptor register name.
func DefaultInterceptorRegisterName() string {
	return defaultInterceptorRegisterName
}

// This Handler will do nothing.
type DefaultInterceptor struct {
}

func (h *DefaultInterceptor) Handle(c *Context) bool {
	return true
}

func (h *DefaultInterceptor) Update(data interface{}) error {
	return nil
}

func (h *DefaultInterceptor) Release() {}

// Create DefaultInterceptor function.
func NewDefaultInterceptor(data interface{}) (Handler, error) {
	return new(DefaultInterceptor), nil
}

// Return DefaultNotFound register name.
func DefaultNotFoundRegisterName() string {
	return defaultNotFoundRegisterName
}

// This Handler return status code 404 and custom message.
type DefaultNotFound struct {
	InterceptData
}

func (h *DefaultNotFound) Handle(c *Context) bool {
	h.InterceptData.WriteToResponse(c.Res)
	return true
}

// Arg data is *InterceptData type.
func (h *DefaultNotFound) Update(data interface{}) error {
	if data != nil {
		d, ok := data.(*InterceptData)
		if !ok {
			return errors.New(`data must be "*InterceptData"`)
		}
		h.InterceptData = *d
	}
	h.InterceptData.Check(http.StatusNotFound)
	return nil
}

func (h *DefaultNotFound) Release() {}

// Create DefaultNotFound function.
func NewDefaultNotFound(data interface{}) (Handler, error) {
	h := new(DefaultNotFound)
	h.Update(data)
	return h, nil
}

type InterceptData struct {
	StatusCode int `json:"statusCode"`
	// Response header["Content-Type"]
	ContentType string `json:"contentType"`
	// Response body string
	Message string `json:"message"`
}

func (d *InterceptData) Check(code int) {
	if d.StatusCode == 0 {
		d.StatusCode = code
	}
	if d.ContentType == "" {
		d.ContentType = mime.TypeByExtension(".html")
	}
}

func (d *InterceptData) WriteToResponse(res http.ResponseWriter) error {
	res.WriteHeader(d.StatusCode)
	res.Header().Add("Content-Type", d.ContentType)
	io.WriteString(res, d.Message)
	return nil
}
