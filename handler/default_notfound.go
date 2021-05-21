package handler

import (
	"io"
	"net/http"
)

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
