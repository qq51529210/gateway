# gateway
使用golang基于标准库net/http开发的网关。

所有的功能，都是通过实现handler.Handler接口来完成的。

## 设计

1. 定义Handler接口，所有的功能都是通过实现接口完成。
2. 在http.ServeHTTP函数中，分成拦截和转发/404两大块，每一块都有Handler的调用链。
3. 添加一个新的功能，只需要调用RegisterHandler()注册，就可以动态生成实例，不需要改动原来的代码。
4. 通过http api接口，可以在运行时修改Handler的调用链，达到增删功能的目的。

## 已实现的功能

- 默认拦截[handler/default_interceptor.go](./handler/ip_interceptor.go)
- 默认转发[handler/default_handler.go](./handler/default_handler.go)
- 默认404[handler/default_notfound.go](./handler/default_notfound.go)
- IP拦截[handler/ip_interceptor.go](./handler/ip_interceptor.go)

## 待实现的功能

- 认证
- 限流
- 熔断
- 调用链起始节点
- 管理
