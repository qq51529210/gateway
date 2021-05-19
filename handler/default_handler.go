package handler

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/qq51529210/gateway/util"
)

var (
	defaultHandlerName = HandlerName(&DefaultHandler{}) // handler的名称
)

// 返回DefaultHandler的注册名称
func DefaultHandlerName() string {
	return defaultHandlerName
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

// Handler接口。
// data的json格式：
// {
// 	"requestUrl": "",
// 	"requestHeader": [
// 		"",
// 		"",
// 		...
// 	],
// 	"requestAdditionHeader": {
// 		"": "",
// 		"": "",
// 		...
// 	},
// 	"requestTimeout": 0,
// 	"responseAdditionHeader": {
// 		"": "",
// 		"": "",
// 		...
// 	}
// }
func (h *DefaultHandler) Update(data interface{}) error {
	initData, ok := data.(map[string]interface{})
	if !ok {
		return errors.New(`data must be "map[string]interface{}"`)
	}
	var newHandler DefaultHandler
	// requestUrl
	str, err := util.Data(initData).String("requestUrl")
	if err != nil {
		return err
	}
	if str != "" {
		newHandler.RequestUrl, err = url.Parse(str)
		if err != nil {
			return fmt.Errorf(`"requestUrl" %s`, err.Error())
		}
	}
	// requestTimeout
	newHandler.RequestTimeout = h.RequestTimeout
	value, ok := initData["requestTimeout"]
	if ok {
		integer, ok := value.(float64)
		if !ok {
			return errors.New(`"requestTimeout" must be "int64" type`)
		}
		newHandler.RequestTimeout = time.Duration(integer) * time.Millisecond
	}
	// requestHeader
	newHandler.RequestHeader = make(map[string]int)
	ss, err := util.Data(initData).StringSlice("requestHeader")
	if err != nil {
		return err
	}
	for _, s := range ss {
		newHandler.RequestHeader[s] = 1
	}
	// requestAdditionHeader
	newHandler.RequestAdditionHeader, err = util.Data(initData).StringMap("requestAdditionHeader")
	if err != nil {
		return err
	}
	// responseAdditionHeader
	newHandler.ResponseAdditionHeader, err = util.Data(initData).StringMap("responseAdditionHeader")
	if err != nil {
		return err
	}
	h.RequestUrl = newHandler.RequestUrl
	if len(newHandler.RequestHeader) > 0 {
		h.RequestHeader = newHandler.RequestHeader
	}
	if len(newHandler.RequestAdditionHeader) > 0 {
		h.RequestAdditionHeader = newHandler.RequestAdditionHeader
	}
	if len(newHandler.ResponseAdditionHeader) > 0 {
		h.ResponseAdditionHeader = newHandler.ResponseAdditionHeader
	}
	return nil
}

// 创建DefaultHandler的函数，data的json格式：
// {
// 	"requestUrl": "",
// 	"requestHeader": [
// 		"",
// 		"",
// 		...
// 	],
// 	"requestAdditionHeader": {
// 		"": "",
// 		"": "",
// 		...
// 	},
// 	"requestTimeout": 0,
// 	"responseAdditionHeader": {
// 		"": "",
// 		"": "",
// 		...
// 	}
// }
// requestUrl表示代理服务的url（必须），requestHeader表示需要转发哪些字段（可选），
// requestAdditionHeader表示额外添加/覆盖原来的字段（可选），requestTimeout表示转发请求超时（毫秒）（可选），
// responseAdditionHeader表示额外添加/覆盖代理服务的响应字段（可选）。
func NewDefaultHandler(data interface{}) (Handler, error) {
	h := new(DefaultHandler)
	err := h.Update(data)
	if err != nil {
		return nil, err
	}
	if h.RequestUrl == nil {
		return nil, errors.New(`"requestUrl" must be defined`)
	}
	return h, nil
}
