package handler

import (
	"fmt"
	"reflect"
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
