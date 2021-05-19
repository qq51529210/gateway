package handler

import (
	"fmt"
	"reflect"

	"github.com/qq51529210/gateway/util"
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

// 创建一个新的Handler的函数，data是Handler初始化的数据。
// 每个Handler的实现都需要使用RegisterHandler()注册。
type NewHandlerFunc func(data map[string]interface{}) (Handler, error)

// 注册创建Handler的函数。
// 做法是在Handler的实现"xxx_handler.go"中的init()注册。
// 参考"./ip_interceptor.go"的实现。
func RegisterHandler(name string, newFunc NewHandlerFunc) {
	handlerFunc[name] = newFunc
}

// 创建Handler，data的json格式：{ "name": "handler", ... }，
// name就是RegisterHandler()的参数，name为空或handlerFunc没有，将生成DefaultHandler。
func NewHandler(data map[string]interface{}) (Handler, error) {
	name, err := util.Data(data).MustString("name")
	if err != nil {
		return nil, err
	}
	newFunc, ok := handlerFunc[name]
	if !ok {
		return NewDefaultHandler(data)
	}
	h, err := newFunc(data)
	if err != nil {
		return nil, fmt.Errorf(`"%s" %s`, name, err.Error())
	}
	return h, nil
}

// 返回h所在的包和结构。在注册时很有用。
// 比如，DefaultHandler返回的是"github.com/qq51529210/gateway/handler.DefaultHandler"。
// 代码编译阶段就可以避免重名的情况。
func HandlerName(h Handler) string {
	_type := reflect.TypeOf(h).Elem()
	return _type.PkgPath() + "." + _type.Name()
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
