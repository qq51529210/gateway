# gateway
使用golang基于标准库net/http开发的网关。

## 设计与使用

1. 主要的逻辑代码如下，所以只要实现了相应步骤的调用链中的Handler就可以了。

```go
func ServeHTTP(...){
  // 第一步，执行拦截的调用链
  Interceptors.Handle()
  // 第二步，如果是转发的路由
  Handlers.Handle()
  // 第二步，如果不匹配路由
  NotFound.Handle()
}
```

2. 添加一个新的功能，只需要调用RegisterHandler()注册，就可以动态生成实例，不需要改动原来的代码。下面是[handler/default_handler.go](./handler/default_handler.go)中的代码片段。

```go
var (
	ipInterceptorRegisterName = HandlerName(&IPInterceptor{}) // 实现的Handler的注册名称
)

func init() {
	RegisterHandler(ipInterceptorRegisterName, NewIPAddrInterceptor) // 注册
}
```

3. 根据数据，动态生成Handler的调用链。

```go
	// 生成网关Gateway实例
	return NewGateway(&NewGatewayData{
    Listen: ":33966",
    Interceptor: []*handler.NewHandlerData{
				{
					Name: Interceptor1, // 实现的Handler的注册名称
					Data: Data1, // Interceptor1初始化的数据，字符串
				},
				{
					Name: handler.IPInterceptorRegisterName(), // 包里实现的IPInterceptor
          Data: Data2, // IPInterceptor初始化的数据，格式看NewIPAddrInterceptor()函数的说明
				},
				{
					Name: Interceptor3,
					Data: Data3,
				},
    },
    NotFound: []*handler.NewHandlerData{
				{
					Name: NotFound1,
					Data: Data1,
				},
				{
					Name: NotFound2,
					Data: Data2,
				},
    },
		Handler: map[string][]*handler.NewHandlerData{
    },
			routue1: { // 路由，比如"/user"
				{
					Name: Handler1,
					Data: Data1,
				},
				{
					Name: handler.DefaultHandlerName(), // 包里实现的DefaultHandler
					Data: Data2, // DefaultHandler初始化的数据，格式看NewDefaultHandler()函数的说明
				},
			},
			routue2: {
				{
					Name: handler.DefaultHandlerName(),
					Data: Data,
				},
			},
		},
	})
```

   

## 已实现的功能

- 默认拦截[DefaultInterceptor.go](./handler/ip_interceptor.go)，什么都不做。
- 默认转发[DefaultHandler.go](./handler/default_handler.go)，可以指定转发哪些header，附加额外的请求和响应header。
- 默认404[DefaultNotfound.go](./handler/default_notfound.go)，返回404和一些文本。
- IP拦截[IPInterceptor.go](./handler/ip_interceptor.go)，拦截指定的IP地址。
- 身份认证拦截[AuthenticationInterceptor.go](./handler/authentication_interceptor.go)，先检查cookie，如果没有再检查Authorization的请求头。拦截成功返回403和文本。

## 待实现的功能

- 限流
- 熔断
- 调用链起始节点
- 管理
