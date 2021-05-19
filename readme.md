# gateway
使用golang基于标准库net/http开发的网关。

## 功能

- 拦截
- 转发
- 认证
- 限流
- 熔断
- 调用链起始节点
- 管理

## 使用

1. 使用URL的顶级级目录作为转发的服务的key。比如，"/user/1"表示将"/1"转发到user服务。
2. 不同的服务，可以有不同的Handler。比如"/user"使用UserHandler，"/goods"使用GoodsHandler。
3. 在运行时可以通过api接口，动态修改Handler。比如"/user"使用UserHandler1，修改成UserHandler2。
4. 插件式Handler，添加一个新的Handler代码，无需修改原有代码。
   1. IP地址黑/白名单过滤。
   2. 