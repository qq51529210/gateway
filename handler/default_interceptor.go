package handler

var (
	defaultInterceptorRegisterName = HandlerName(&DefaultInterceptor{}) // 注册名称
)

func init() {
	RegisterHandler(defaultInterceptorRegisterName, NewDefaultInterceptor) // 注册
}

// 获取注册名称
func DefaultInterceptorRegisterName() string {
	return defaultInterceptorRegisterName
}

// 什么都不做
type DefaultInterceptor struct {
}

// 实现接口
func (h *DefaultInterceptor) Handle(c *Context) bool {
	return true
}

// 实现接口
func (h *DefaultInterceptor) Update(data interface{}) error {
	return nil
}

// 实现接口
func (h *DefaultInterceptor) Name() string {
	return defaultInterceptorRegisterName
}

func NewDefaultInterceptor(data *NewHandlerData) (Handler, error) {
	return new(DefaultInterceptor), nil
}
