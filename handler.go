package gateway

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/qq51529210/log"
)

var (
	handlerFunc = make(map[string]NewHandlerFunc) // 创建处理器的函数的表
	contextPool = new(sync.Pool)
)

func init() {
	contextPool.New = func() interface{} {
		return new(Context)
	}
}

// 调用链中传递的上下文数据
type Context struct {
	Res  http.ResponseWriter
	Req  *http.Request
	Data interface{} // 传递数据用的
}

// 处理接口
type Handler interface {
	// 返回false表示失败，终止调用链
	Handle(*Context) bool
}

// 创建一个新的拦截器的函数，data是Handler初始化的数据。
type NewHandlerFunc func(data interface{}) (Handler, error)

// 注册创建Handler的函数，原理和标准库"db/sql"一致。
// 在其他开发包的init()注册其实现的Handler，就可以通过name动态的创建。
func RegisterHandler(name string, newFunc NewHandlerFunc) {
	handlerFunc[name] = newFunc
}

// 根据data和已经注册的NewHandlerFunc，创建相应的Handler。
func NewHandler(name string, data interface{}) (Handler, error) {
	newFunc, ok := handlerFunc[name]
	if !ok {
		return NewDefaultHandler(data)
	}
	return newFunc(data)
}

// 默认处理，只是转发
type DefaultHandler struct {
	RoutePath              string            // 注册的路由路径
	RequestUrl             *url.URL          // 转发请求的服务地址
	RequestTimeout         time.Duration     // 转发请求超时，单位毫秒
	RequestHeader          map[string]int    // 转发请求header
	RequestAdditionHeader  map[string]string // 转发请求附加的header
	ResponseAdditionHeader map[string]string // 转发相应附加的header
}

// 接口实现
func (h *DefaultHandler) Handle(c *Context) bool {
	// 转发的url
	var request http.Request
	request.Method = c.Req.Method
	request.URL = new(url.URL)
	*request.URL = *h.RequestUrl
	request.URL.Path = c.Req.URL.Path[len(h.RoutePath):]
	request.ContentLength = c.Req.ContentLength
	// 提取指定转发的header
	for k := range h.RequestHeader {
		v := c.Req.Header.Get(k)
		if v != "" {
			request.Header.Set(k, v)
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
		log.Error(err)
		return false
	}
	// 转发结果
	c.Res.WriteHeader(response.StatusCode)
	header := c.Res.Header()
	for k, v := range response.Header {
		for _, s := range v {
			header.Add(k, s)
		}
	}
	for k, v := range h.ResponseAdditionHeader {
		header.Add(k, v)
	}
	io.Copy(c.Res, response.Body)
	return true
}

// 创建DefaultHandler的函数，data的json格式
// {
// 	"routePath": "",
// 	"requestUrl": "",
// 	"requestHeader": [
// 		"name1",
// 		"name2",
// 		...
// 	],
// 	"requestAdditionHeader": {
// 		"name1": "value1",
// 		"name2": "value2",
// 		...
// 	},
// 	"requestTimeout": 2000,
// 	"responseAdditionHeader": {
// 		"name1": "value1",
// 		"name2": "value2",
// 		...
// 	}
// }
func NewDefaultHandler(data interface{}) (Handler, error) {
	initData, ok := data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(`init data must be "map[string]interface{}"`)
	}
	var err error
	h := new(DefaultHandler)
	h.RequestHeader = make(map[string]int)
	h.RequestAdditionHeader = make(map[string]string)
	h.ResponseAdditionHeader = make(map[string]string)
	// RoutePath
	h.RoutePath, err = MustGetString(initData, "routePath")
	if err != nil {
		return nil, err
	}
	// RequestUrl
	str, err := MustGetString(initData, "requestUrl")
	if err != nil {
		return nil, err
	}
	h.RequestUrl, err = url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf(`"requestUrl" %s`, err.Error())
	}
	// RequestTimeout
	value, ok := initData["requestTimeout"]
	if ok {
		integer, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf(`"requestTimeout" must be "int64"`)
		}
		h.RequestTimeout = time.Duration(integer) * time.Millisecond
	}
	// RequestHeader
	value, ok = initData["requestHeader"]
	if ok {
		a, ok := value.([]interface{})
		if !ok {
			return nil, fmt.Errorf(`"requestHeader" must be "[]string"`)
		}
		for i, v := range a {
			str, ok = v.(string)
			if !ok {
				return nil, fmt.Errorf(`"requestHeader" item[%d] must be string`, i)
			}
			h.RequestHeader[str] = 1
		}
	}
	// RequestAdditionHeader
	value, ok = initData["requestAdditionHeader"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(`"requestAdditionHeader" must be "map[string]string"`)
		}
		for k, v := range m {
			str, ok = v.(string)
			if !ok {
				return nil, fmt.Errorf(`"requestAdditionHeader"."%s" value must be string`, k)
			}
			h.RequestAdditionHeader[k] = str
		}
	}
	// ResponseAdditionHeader
	value, ok = initData["ResponseAdditionHeader"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(`"ResponseAdditionHeader" must be "map[string]string"`)
		}
		for k, v := range m {
			str, ok = v.(string)
			if !ok {
				return nil, fmt.Errorf(`"ResponseAdditionHeader"."%s" value must be string`, k)
			}
			h.ResponseAdditionHeader[k] = str
		}
	}
	// 返回
	return h, nil
}

// 默认全局拦截，什么也不做
type DefaultInterceptor struct {
}

// 接口实现
func (h *DefaultInterceptor) Handle(c *Context) bool {
	return true
}

// 默认匹配失败处理，返回404
type DefaultNotFound struct {
}

// 接口实现
func (h *DefaultNotFound) Handle(c *Context) bool {
	c.Res.WriteHeader(http.StatusNotFound)
	return true
}
