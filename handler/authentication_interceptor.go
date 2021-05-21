package handler

var (
	authenticationInterceptorRegisterName = HandlerName(&AuthenticationInterceptor{}) // 注册名称
)

func init() {
	RegisterHandler(authenticationInterceptorRegisterName, NewAuthenticationInterceptor) // 注册
}

// 获取AuthenticationInterceptor的注册名称
func AuthenticationInterceptorRegisterName() string {
	return ipInterceptorRegisterName
}

// 认证拦截处理
type AuthenticationInterceptor struct {
}

// 实现接口
func (h *AuthenticationInterceptor) Handle(c *Context) bool {
	return true
}

// 实现接口
func (h *AuthenticationInterceptor) Update(data interface{}) error {
	return nil
}

// 实现接口
func (h *AuthenticationInterceptor) Name() string {
	return ipInterceptorRegisterName
}

func NewAuthenticationInterceptor(data *NewHandlerData) (Handler, error) {
	h := new(AuthenticationInterceptor)

	return h, nil
}
