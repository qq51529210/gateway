package gateway

import "net/http"

var (
	interceptor = make(map[string]NewInterceptorFunc) // 创建拦截器的函数的表
)

func init() {
	interceptor[""] = NewDefaultInterceptor
	interceptor["default"] = NewDefaultInterceptor
}

// 注册创建Interceptor的函数，原理和标准库"db/sql"一致
func RegisterInterceptor(_type string, _new NewInterceptorFunc) {
	interceptor[_type] = _new
}

// 根据data创建相应的Interceptor
// data的json格式
// 	{
// 		"type":"",
//		...
// 	}
func NewInterceptor(data map[string]interface{}) (Interceptor, error) {
	if data == nil {
		return new(DefaultInterceptor), nil
	}
	str, err := hasString(data, "type")
	if err != nil {
		return nil, err
	}
	_new, ok := interceptor[str]
	if !ok {
		return new(DefaultInterceptor), nil
	}
	return _new(data)
}

// 拦截接口
type Interceptor interface {
	Intercept(res http.ResponseWriter, req *http.Request) bool
}

// 创建一个新的拦截器的函数，m是初始化的参数，返回具体的实现，或者错误
type NewInterceptorFunc func(m map[string]interface{}) (Interceptor, error)

// 默认拦截器，什么也不做
type DefaultInterceptor struct {
}

func (di *DefaultInterceptor) Intercept(res http.ResponseWriter, req *http.Request) bool {
	return false
}

// 创建默认的拦截器
func NewDefaultInterceptor(m map[string]interface{}) (Interceptor, error) {
	return new(DefaultInterceptor), nil
}
