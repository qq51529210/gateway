package gateway

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/qq51529210/log"
)

var (
	// 创建处理器的函数的表
	handler = make(map[string]NewHandlerFunc)
)

func init() {
	handler[""] = NewDefaultHandler
	handler["default"] = NewDefaultHandler
}

// 注册创建处理器的函数，原理和标准库"db/sql"一致
func RegisterHandler(_type string, _new NewHandlerFunc) {
	handler[_type] = _new
}

// 根据data创建相应的处理器
// data的json结构如下
// 	{
// 		"routePath1":{
// 			"type":"",
// 			...
// 		},
// 		"routePath2":{
// 			"type":"",
// 			...
// 		}
// 	}
func NewHandler(data map[string]interface{}) (http.Handler, error) {
	str, err := hasString(data, "type")
	if err != nil {
		return nil, err
	}
	_new, ok := handler[str]
	if !ok {
		return NewDefaultHandler(data)
	}
	return _new(data)
}

// 创建一个新的拦截器的函数，data是初始化的参数，返回具体的实现，或者错误
type NewHandlerFunc func(data map[string]interface{}) (http.Handler, error)

// 默认处理，只是转发
type DefaultHandler struct {
	RoutePath             string            // 网关的路径
	ForwardUrl            *url.URL          // 代理的服务地址
	ForwardHeader         map[string]int    // 需要继续传递的header名称
	ForwardAdditionHeader map[string]string // 附加的header
	RequestTimeout        time.Duration     // 调用代理的服务超时，单位毫秒
}

func (h *DefaultHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var request http.Request
	request.Method = req.Method
	request.URL = new(url.URL)
	*request.URL = *h.ForwardUrl
	request.URL.Path = path.Join(req.URL.Path[len(h.RoutePath):])
	request.ContentLength = req.ContentLength
	// 需要转发的header
	for k := range h.ForwardHeader {
		v := req.Header.Get(k)
		if v != "" {
			request.Header.Set(k, v)
		}
	}
	// 附加的header
	for k, v := range h.ForwardAdditionHeader {
		request.Header.Set(k, v)
	}
	request.Body = req.Body
	// 发送
	client := &http.Client{Timeout: h.RequestTimeout}
	response, err := client.Do(&request)
	if err != nil {
		log.Error(err)
		return
	}
	// 提取头部
	header := res.Header()
	for k, v := range response.Header {
		for _, s := range v {
			header.Add(k, s)
		}
	}
	res.WriteHeader(response.StatusCode)
	io.Copy(res, response.Body)
}

// 根据data创建相应的Handler
// data的json格式
// {
// 	"routePath": "",
// 	"forwardUrl": "",
// 	"forwardHeader": ["",""],
// 	"forwardAdditionHeader": {
//	 "":"",
//	 "":""
// 	},
// 	"requestTimeout": 0
// }
func NewDefaultHandler(data map[string]interface{}) (http.Handler, error) {
	var err error
	h := new(DefaultHandler)
	h.ForwardHeader = make(map[string]int)
	h.ForwardAdditionHeader = make(map[string]string)
	// routePath
	h.RoutePath, err = getString(data, "routePath")
	if err != nil {
		return nil, err
	}
	// forwardUrl
	var str string
	str, err = getString(data, "forwardUrl")
	if err != nil {
		return nil, err
	}
	_url, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf(`"forwardUrl" %s"`, err.Error())
	}
	h.ForwardUrl = _url
	// forwardHeader
	val, ok := data["forwardHeader"]
	if ok {
		a, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf(`"forwardHeader" must be "[]string"`)
		}
		for i, v := range a {
			str, ok = v.(string)
			if !ok {
				return nil, fmt.Errorf(`"forwardHeader" item[%d] must be string`, i)
			}
			h.ForwardHeader[str] = 1
		}
	}
	// forwardAdditionHeader
	val, ok = data["forwardAdditionHeader"]
	if ok {
		a, ok := val.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(`"forwardAdditionHeader" must be "map[string]string"`)
		}
		for k, v := range a {
			str, ok = v.(string)
			if !ok {
				return nil, fmt.Errorf(`"forwardAdditionHeader"."%s" value must be string`, k)
			}
			h.ForwardAdditionHeader[k] = str
		}
	}
	// requestTimeout
	val, ok = data["requestTimeout"]
	if ok {
		a, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf(`"forwardAdditionHeader"  must be "int64"`)
		}
		h.RequestTimeout = time.Duration(a) * time.Millisecond
	}
	// 返回
	return h, nil
}
