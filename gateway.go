package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"sync"

	"github.com/qq51529210/gateway/handler"
	"github.com/qq51529210/gateway/util"
)

var (
	contextPool = new(sync.Pool) // Context缓存池
)

func init() {
	contextPool.New = func() interface{} {
		return new(handler.Context)
	}
}

// 创建Gateway实例，data的json格式
// {
// 	"listen": "",
// 	"x509CertPEM": "",
// 	"x509KeyPem": "",
// 	"apiListen": "",
// 	"apiX509CertPem": "",
// 	"apiX509KeyPem": "",
// 	"apiAccessToken": "",
// 	"interceptor": [
// 		"handle1": {...},
// 		"handle2": {...},
// 		...
// 	],
// 	"notfound": [
// 		"handle1": {...},
// 		"handle2": {...},
//		...
// 	],
// 	"handler": {
// 		"/path1": [
// 			"handle1": {...},
// 			"handle2": {...},
//			...
// 		],
// 		"/path2": [
// 			"handle1": {...},
// 			"handle3": {...},
//			...
// 		],
// 		"/path3": [
// 			"handle3": {...},
//			...
// 		]
// 	}
// }
func NewGateway(data map[string]interface{}) (*Gateway, error) {
	var err error
	gw := new(Gateway)
	// 	"listen": "",
	gw.Listen, err = util.Data(data).MustString("listen")
	if err != nil {
		return nil, err
	}
	if gw.Listen == "" {
		return nil, errors.New(`"listen" data must not be empty`)
	}
	// 	"x509CertPem": "",
	gw.X509CertPEM, err = util.Data(data).String("x509CertPEM")
	if err != nil {
		return nil, err
	}
	// 	"x509KeyPem": "",
	gw.X509KeyPEM, err = util.Data(data).String("x509KeyPEM")
	if err != nil {
		return nil, err
	}
	// 	"apiListen": "",
	gw.ApiListen, err = util.Data(data).String("apiListen")
	if err != nil {
		return nil, err
	}
	// 	"apiX509CertPem": "",
	gw.ApiX509CertPEM, err = util.Data(data).String("apiX509CertPem")
	if err != nil {
		return nil, err
	}
	// 	"apiX509KeyPem": "",
	gw.ApiX509KeyPEM, err = util.Data(data).String("apiX509KeyPem")
	if err != nil {
		return nil, err
	}
	// 	"apiAccessToken": "",
	gw.ApiAccessToken, err = util.Data(data).String("apiAccessToken")
	if err != nil {
		return nil, err
	}
	// 	"interceptor": [
	// 		{
	//			"name": "interceptor1",
	//			...
	//		},
	// 		{
	//			"name": "interceptor2",
	//		},
	//		...
	// 	],
	value, ok := data["interceptor"]
	if ok {
		err = gw.NewInterceptor(value)
		if err != nil {
			return nil, fmt.Errorf(`"interceptor" %s`, err.Error())
		}
	}
	// 	"notfound": [
	// 		{
	//			"name": "notfound1",
	//			...
	//		},
	// 		{
	//			"name": "notfound2",
	//		},
	//		"notfound3",
	//		...
	// 	],
	value, ok = data["notfound"]
	if ok {
		err = gw.NewNotFound(value)
		if err != nil {
			return nil, fmt.Errorf(`"notfound" %s`, err.Error())
		}
	}
	// 	"handler": {
	// 		"/path1": [
	// 			{
	//				"name": "handler1",
	// 				...
	// 			},
	// 			{
	//				"name": "handler2",
	// 				...
	// 			},
	//			...
	// 		],
	// 		"/path2": [
	// 			{
	//				"name": "handler2",
	// 				...
	// 			},
	// 			"handler1",
	//			...
	// 		],
	// 		"/path3": [
	// 			"handler3",
	//			...
	// 		]
	//	}
	value, ok = data["handler"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.New(`"handler" must be "map[string]interface{}" type`)
		}
		for k, v := range m {
			err = gw.NewHandler(k, v)
			if err != nil {
				return nil, fmt.Errorf(`"handler"."%s" %s`, k, err.Error())
			}
		}
	}
	return gw, nil
}

// 网关app
type Gateway struct {
	Listen         string            // 监听地址
	X509CertPEM    string            // X509证书公钥，base64
	X509KeyPEM     string            // X509证书私钥，base64
	ApiListen      string            // api服务监听地址
	ApiX509CertPEM string            // api服务X509证书公钥，base64
	ApiX509KeyPEM  string            // api服务X509证书私钥，base64
	ApiAccessToken string            // api服务访问token
	Interceptor    []handler.Handler // 全局拦截
	NotFound       []handler.Handler // 匹配失败
	handler        sync.Map          // 处理函数路由表，string:[]Handler
	server         http.Server       // http包Server
}

// 开始服务
func (gw *Gateway) Serve() error {
	// 检查
	if len(gw.Interceptor) == 0 {
		gw.Interceptor = []handler.Handler{new(handler.DefaultInterceptor)}
	}
	if len(gw.NotFound) == 0 {
		gw.NotFound = []handler.Handler{new(handler.DefaultNotFound)}
	}
	// 初始化服务
	gw.server.Handler = http.HandlerFunc(gw.handlerHTTP)
	listener, err := net.Listen("tcp", gw.Listen)
	if err != nil {
		return nil
	}
	// 如果有证书，就使用tls
	if gw.X509CertPEM != "" && gw.X509KeyPEM != "" {
		certificate, err := tls.X509KeyPair([]byte(gw.X509CertPEM), []byte(gw.X509KeyPEM))
		if err != nil {
			return nil
		}
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{certificate},
		})
	}
	// 监听
	return gw.server.Serve(listener)
}

// 关闭服务
func (gw *Gateway) Close() error {
	return gw.server.Close()
}

// 处理请求
func (gw *Gateway) handlerHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := contextPool.Get().(*handler.Context)
	ctx.Req = req
	ctx.Res = res
	ctx.Data = nil
	// 拦截
	ctx.Interceptor = gw.Interceptor
	for _, h := range ctx.Interceptor {
		if !h.Handle(ctx) {
			contextPool.Put(ctx)
			return
		}
	}
	// 获取handler
	ctx.Path = util.TopDir(req.URL.Path)
	value, ok := gw.handler.Load(ctx.Path)
	if !ok {
		ctx.NotFound = gw.NotFound
		for _, h := range ctx.NotFound {
			if !h.Handle(ctx) {
				break
			}
		}
		contextPool.Put(ctx)
		return
	}
	// 处理
	ctx.Handler = value.([]handler.Handler)
	for _, h := range ctx.Handler {
		if !h.Handle(ctx) {
			contextPool.Put(ctx)
			return
		}
	}
	contextPool.Put(ctx)
}

// 设置处理，每个route对应一组handler
func (gw *Gateway) SetHandler(route string, handler ...handler.Handler) {
	gw.handler.Store(util.TopDir(path.Clean("/"+route)), handler)
}

// 创建并设置新的处理，data的json格式
// [
// 		{
//			"name": "handler1",
//			...
//		},
// 		{
//			"name": "handler2",
//			...
//		},
//		...
// ]
func (gw *Gateway) NewHandler(route string, data interface{}) error {
	handler, err := newHandler("handler", data)
	if err != nil {
		return err
	}
	gw.SetHandler(route, handler...)
	return nil
}

// 创建并设置新的拦截，如果没有生成新的Handler，则不生效。
func (gw *Gateway) NewInterceptor(data interface{}) error {
	handler, err := newHandler("iterceptor", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.Interceptor = handler
	}
	return nil
}

// 设置新的404，如果没有生成新的Handler，则不生效。
func (gw *Gateway) NewNotFound(data interface{}) error {
	handler, err := newHandler("notFound", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.NotFound = handler
	}
	return nil
}

func newHandler(name string, data interface{}) ([]handler.Handler, error) {
	array, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf(`"%s" data must be "[]interface{}" type`, name)
	}
	newHD := make([]handler.Handler, 0)
	for i, a := range array {
		v, ok := a.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(`"%s[%d]" data must be "map[string]interface{}" type`, name, i)
		}
		hd, err := handler.NewHandler(v)
		if err != nil {
			return nil, fmt.Errorf(`"%s[%d]" %s`, name, i, err.Error())
		}
		newHD = append(newHD, hd)
	}
	return newHD, nil
}
