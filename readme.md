# gateway
一个http的网关雏形，主要有以下功能。
- 拦截

  1. [Interceptor](./interceptro.go)，接口的定义。

  1. [DefaultInterceptor](./interceptro.go)，默认的拦截器，什么都不拦截。

  1. [IPAddrInterceptor](./ip_interceptro.go)，ip黑名单拦截。

- 转发

  1. [DefaultHandler](./handler.go)，默认的处理，简单的转发请求。

## 使用

