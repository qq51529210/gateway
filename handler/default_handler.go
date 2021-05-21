package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

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
