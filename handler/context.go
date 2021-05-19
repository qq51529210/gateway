package handler

import (
	"net/http"
)

// 调用链中传递的上下文数据
type Context struct {
	Res         http.ResponseWriter
	Req         *http.Request
	Path        string      // 匹配的路由的path
	Data        interface{} // 用于调用链之间传递临时数据
	Interceptor []Handler   // Gateway当前的所有Interceptor，任意时刻有效
	NotFound    []Handler   // Gateway当前的所有NotFound，匹配到Handler无效
	Handler     []Handler   // Gateway当前的所有Handler，匹配到NotFound无效
}
