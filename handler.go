package gateway

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

var (
	handlerFunc            = make(map[string]NewHandlerFunc)      // 创建处理器的函数的表
	contextPool            = new(sync.Pool)                       // Context缓存
	defaultHandlerName     = HandlerName(new(DefaultHandler))     // 注册名称
	defaultInterceptorName = HandlerName(new(DefaultInterceptor)) // 注册名称
	defaultNotFoundName    = HandlerName(new(DefaultNotFound))    // 注册名称
)

func init() {
	contextPool.New = func() interface{} {
		return new(Context)
	}
}

// 调用链中传递的上下文数据
type Context struct {
	Res         http.ResponseWriter
	Req         *http.Request
	Path        string      // 匹配的路由的path
	Data        interface{} // 用于调用链之间传递临时数据
	Interceptor []Handler   // Gateway当前的所有Interceptor，任意时刻有效
	NotFound    []Handler   // Gateway当前的所有NotFound，匹配到Handler无效
	Handler     []Handler   // Gateway当前的所有Handler，匹配到NotFound无效
}

// 处理接口
type Handler interface {
	// 返回false表示失败，终止调用链
	Handle(*Context) bool
	// 更新自身数据
	Update(interface{}) error
	// 返回注册的名称
	Name() string
}

// 创建一个新的拦截器的函数，data是Handler初始化的数据。
type NewHandlerFunc func(data interface{}) (Handler, error)

// 注册创建Handler的函数，原理和标准库"db/sql"一致。
// 在其他开发包的init()注册其实现的Handler，就可以通过name动态的创建。
func RegisterHandler(name string, newFunc NewHandlerFunc) {
	handlerFunc[name] = newFunc
}

// 创建Handler，data的json格式
// {
//	"name": "handler",
// 	...
// },
// 或者
// "handler",
func NewHandler(data interface{}) (Handler, error) {
	var name string
	switch v := data.(type) {
	case string:
		name = v
	case map[string]interface{}:
		var err error
		name, err = GetString(v, "name")
		if err != nil {
			return nil, err
		}
	default:
		return nil, errors.New(`data must be "string" or "map[string]interface{}" type`)
	}
	newFunc, ok := handlerFunc[name]
	if !ok {
		return NewDefaultHandler(data)
	}
	return newFunc(data)
}

// 默认处理，只是转发
type DefaultHandler struct {
	RequestUrl             *url.URL          // 转发请求的服务地址
	RequestTimeout         time.Duration     // 转发请求超时，单位毫秒
	RequestHeader          map[string]int    // 转发请求header
	RequestAdditionHeader  map[string]string // 转发请求附加的header
	ResponseAdditionHeader map[string]string // 转发相应附加的header
}

func (h *DefaultHandler) Name() string {
	return defaultHandlerName
}

// 接口实现
func (h *DefaultHandler) Handle(c *Context) bool {
	// 转发的url
	var request http.Request
	request.Method = c.Req.Method
	request.URL = new(url.URL)
	*request.URL = *h.RequestUrl
	request.URL.Path = c.Req.URL.Path[len(c.Path):]
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
		fmt.Println(err)
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

// 接口实现，data的json格式
// {
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
func (h *DefaultHandler) Update(data interface{}) error {
	initData, ok := data.(map[string]interface{})
	if !ok {
		return errors.New(`data must be "map[string]interface{}"`)
	}
	// RequestUrl
	str, err := GetString(initData, "requestUrl")
	if err != nil {
		return err
	}
	if str != "" {
		h.RequestUrl, err = url.Parse(str)
		if err != nil {
			return fmt.Errorf(`"requestUrl" %s`, err.Error())
		}
	}
	return h.update(initData)
}

// Update和NewDefaultHandler公共函数
func (h *DefaultHandler) update(data map[string]interface{}) error {
	// RequestTimeout
	value, ok := data["requestTimeout"]
	if ok {
		integer, ok := value.(float64)
		if !ok {
			return errors.New(`"requestTimeout" must be "int64"`)
		}
		h.RequestTimeout = time.Duration(integer) * time.Millisecond
	}
	// RequestHeader
	value, ok = data["requestHeader"]
	if ok {
		a, ok := value.([]interface{})
		if !ok {
			return errors.New(`"requestHeader" must be "[]string"`)
		}
		for i, v := range a {
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf(`"requestHeader" item[%d] must be string`, i)
			}
			h.RequestHeader[str] = 1
		}
	}
	// RequestAdditionHeader
	value, ok = data["requestAdditionHeader"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return errors.New(`"requestAdditionHeader" must be "map[string]string"`)
		}
		for k, v := range m {
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf(`"requestAdditionHeader"."%s" value must be string`, k)
			}
			h.RequestAdditionHeader[k] = str
		}
	}
	// ResponseAdditionHeader
	value, ok = data["responseAdditionHeader"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return errors.New(`"responseAdditionHeader" must be "map[string]string"`)
		}
		for k, v := range m {
			str, ok := v.(string)
			if !ok {
				return fmt.Errorf(`"responseAdditionHeader"."%s" value must be string`, k)
			}
			h.ResponseAdditionHeader[k] = str
		}
	}
	return nil
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
	// RequestUrl
	str, err := MustGetString(initData, "requestUrl")
	if err != nil {
		return nil, err
	}
	h.RequestUrl, err = url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf(`"requestUrl" %s`, err.Error())
	}
	err = h.update(initData)
	if err != nil {
		return nil, err
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

// 接口实现
func (h *DefaultInterceptor) Update(data interface{}) error {
	return nil
}

// 接口实现
func (h *DefaultInterceptor) Name() string {
	return defaultInterceptorName
}

// 默认匹配失败处理，返回404
type DefaultNotFound struct {
}

// 接口实现
func (h *DefaultNotFound) Handle(c *Context) bool {
	c.Res.WriteHeader(http.StatusNotFound)
	return true
}

// 接口实现
func (h *DefaultNotFound) Update(data interface{}) error {
	return nil
}

// 接口实现
func (h *DefaultNotFound) Name() string {
	return defaultNotFoundName
}
