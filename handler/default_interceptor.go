package handler

var (
	defaultInterceptorName = HandlerName(&DefaultInterceptor{}) // handler的名称
)

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
