package handler

import "net/http"

var (
	defaultNotFoundName = HandlerName(&DefaultNotFound{}) // handler的名称
)

// 默认匹配失败处理，返回404
type DefaultNotFound struct {
}

// Handler接口
func (h *DefaultNotFound) Handle(c *Context) bool {
	c.Res.WriteHeader(http.StatusNotFound)
	return true
}

// Handler接口
func (h *DefaultNotFound) Update(data interface{}) error {
	return nil
}

// Handler接口
func (h *DefaultNotFound) Name() string {
	return defaultNotFoundName
}
