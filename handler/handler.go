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
	handlerFunc = make(map[string]NewHandlerFunc) // 创建处理器的函数的表
)

// 处理接口
type Handler interface {
	// 返回false表示失败，终止调用链
	Handle(*Context) bool
	// 更新自身数据
	Update(interface{}) error
	// 返回注册的名称
	Name() string
}

// NewHandler使用的参数
type NewHandlerData struct {
	Name string `json:"name"` // 注册的名称，NewHandler使用
	Data string `json:"data"` // 初始化数据，Handler的实现使用
}

// 创建一个新的Handler的函数，data是Handler初始化的数据。
// 每个Handler的实现都需要使用RegisterHandler()注册。
type NewHandlerFunc func(data *NewHandlerData) (Handler, error)

// 注册创建Handler的函数。
// 做法是在Handler的实现"xxx_handler.go"中的init()注册。
func RegisterHandler(name string, newFunc NewHandlerFunc) {
	handlerFunc[name] = newFunc
}

// 创建Handler，data.Name为空，或者没有注册，将生成DefaultHandler。
func NewHandler(data *NewHandlerData) (Handler, error) {
	newFunc, ok := handlerFunc[data.Name]
	if !ok {
		return NewDefaultHandler(data)
	}
	h, err := newFunc(data)
	if err != nil {
		return nil, fmt.Errorf(`"%s" %s`, data.Name, err.Error())
	}
	return h, nil
}

// 返回h所在的包和结构。在注册时很有用。
// 比如，DefaultHandler返回的是"github.com/qq51529210/gateway/handler/DefaultHandler"。
// 代码编译阶段就可以避免重名的情况。
func HandlerName(h Handler) string {
	_type := reflect.TypeOf(h).Elem()
	return _type.PkgPath() + "/" + _type.Name()
}

// 检查handler长度和nil
func CheckHandlers(handler ...Handler) error {
	if len(handler) < 1 {
		return fmt.Errorf("empty handler slice")
	}
	for i, h := range handler {
		if h == nil {
			return fmt.Errorf("handler[%d] is nil", i)
		}
	}
	return nil
}

var (
	defaultHandlerName = HandlerName(&DefaultHandler{}) // handler的名称
)

// 返回DefaultHandler的注册名称
func DefaultHandlerName() string {
	return defaultHandlerName
}

// DefaultHandler初始化数据
type DefaultHandlerData struct {
	RequestUrl             string            `json:"requestUrl"`             // 转发请求的服务地址，必须
	RequestTimeout         int               `json:"requestTimeout"`         // 转发请求超时，单位毫秒，可选
	RequestHeader          []string          `json:"requestHeader"`          // 转发请求header，可选
	RequestAdditionHeader  map[string]string `json:"requestAdditionHeader"`  // 转发请求附加的header，可选
	ResponseAdditionHeader map[string]string `json:"responseAdditionHeader"` // 转发相应附加的header，可选
}

// 默认处理，只是转发
type DefaultHandler struct {
	RequestUrl             *url.URL          // 转发请求的服务地址
	RequestTimeout         time.Duration     // 转发请求超时，单位毫秒
	RequestHeader          map[string]int    // 转发请求header
	RequestAdditionHeader  map[string]string // 转发请求附加的header
	ResponseAdditionHeader map[string]string // 转发相应附加的header
}

// Handler接口
func (h *DefaultHandler) Name() string {
	return defaultHandlerName
}

// Handler接口
func (h *DefaultHandler) Handle(c *Context) bool {
	// 转发的url
	var request http.Request
	request.Method = c.Req.Method
	request.URL = new(url.URL)
	*request.URL = *h.RequestUrl
	request.URL.Path = c.Req.URL.Path[len(c.Path):]
	request.URL.RawQuery = c.Req.URL.RawQuery
	request.URL.RawFragment = c.Req.URL.RawFragment
	request.ContentLength = c.Req.ContentLength
	// 提取指定转发的header
	request.Header = make(http.Header)
	if len(h.RequestHeader) < 1 {
		for k := range c.Req.Header {
			request.Header.Set(k, c.Req.Header.Get(k))
		}
	} else {
		for k := range h.RequestHeader {
			v := c.Req.Header.Get(k)
			if v != "" {
				request.Header.Set(k, v)
			}
		}
	}
	// 附加的header
	for k, v := range h.RequestAdditionHeader {
		request.Header.Set(k, v)
	}
	request.Body = c.Req.Body
	// 转发请求
	client := &http.Client{Timeout: h.RequestTimeout}
	response, err := client.Do(&request)
	if err != nil {
		fmt.Println(err)
		return false
	}
	// 转发结果
	header := c.Res.Header()
	for k, v := range response.Header {
		for _, s := range v {
			header.Add(k, s)
		}
	}
	for k, v := range h.ResponseAdditionHeader {
		header.Add(k, v)
	}
	c.Res.WriteHeader(response.StatusCode)
	io.Copy(c.Res, response.Body)
	return true
}

// Handler接口，data是*DefaultHandlerData
func (h *DefaultHandler) Update(data interface{}) error {
	d, ok := data.(*DefaultHandlerData)
	if !ok {
		return errors.New(`data must be "*DefaultHandlerData"`)
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

// 创建DefaultHandler的函数，已经注册。data的格式为
// {
// 	"name": "github.com/qq51529210/gateway/handler/DefaultHandler",
// 	"data": "DefaultHandlerData的json字符串"
// }
func NewDefaultHandler(data *NewHandlerData) (Handler, error) {
	var d DefaultHandlerData
	err := json.Unmarshal([]byte(data.Data), &d)
	if err != nil {
		return nil, err
	}
	h := new(DefaultHandler)
	err = h.Update(&d)
	if err != nil {
		return nil, err
	}
	if h.RequestUrl == nil {
		return nil, errors.New(`"requestUrl" must be defined`)
	}
	return h, nil
}

var (
	defaultInterceptorRegisterName = HandlerName(&DefaultInterceptor{}) // 注册名称
)

func init() {
	RegisterHandler(defaultInterceptorRegisterName, NewDefaultInterceptor) // 注册
}

// 获取注册名称
func DefaultInterceptorRegisterName() string {
	return defaultInterceptorRegisterName
}

// 什么都不做
type DefaultInterceptor struct {
}

// 实现接口
func (h *DefaultInterceptor) Handle(c *Context) bool {
	return true
}

// 实现接口
func (h *DefaultInterceptor) Update(data interface{}) error {
	return nil
}

// 实现接口
func (h *DefaultInterceptor) Name() string {
	return defaultInterceptorRegisterName
}

func NewDefaultInterceptor(data *NewHandlerData) (Handler, error) {
	return new(DefaultInterceptor), nil
}

var (
	defaultNotFoundRegisterName = HandlerName(&DefaultNotFound{}) // handler的名称
)

func init() {
	RegisterHandler(defaultNotFoundRegisterName, NewDefaultNotFound) // 注册
}

// 获取注册名称
func DefaultNotFoundRegisterName() string {
	return defaultNotFoundRegisterName
}

// 默认匹配失败处理，返回404
type DefaultNotFound struct {
	Data string
}

// Handler接口
func (h *DefaultNotFound) Handle(c *Context) bool {
	c.Res.WriteHeader(http.StatusNotFound)
	io.WriteString(c.Res, h.Data)
	return true
}

// Handler接口
func (h *DefaultNotFound) Update(data interface{}) error {
	s, ok := data.(string)
	if ok && s != "" {
		h.Data = s
	}
	return nil
}

// Handler接口
func (h *DefaultNotFound) Name() string {
	return defaultNotFoundRegisterName
}

// 创建DefaultNotFound，已经注册。data的格式为
// {
// 	"name": "github.com/qq51529210/gateway/handler/DefaultNotFound",
// 	"data": "html文本"
// }
func NewDefaultNotFound(data *NewHandlerData) (Handler, error) {
	h := new(DefaultNotFound)
	h.Update(data.Data)
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
