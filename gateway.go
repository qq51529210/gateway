package gateway

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path"
	"sync"
)

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
// 		{
//			"name": "interceptor1",
//			...
//		},
// 		{
//			"name": "interceptor2",
//		},
//		"interceptor3",
//		...
// 	],
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
// }
func NewGateway(data map[string]interface{}) (*Gateway, error) {
	var err error
	gw := new(Gateway)
	// 	"listen": "",
	gw.Listen, err = MustGetString(data, "listen")
	if err != nil {
		return nil, err
	}
	if gw.Listen == "" {
		return nil, errors.New(`"listen" data must not be empty`)
	}
	// 	"x509CertPem": "",
	gw.X509CertPEM, err = GetString(data, "x509CertPEM")
	if err != nil {
		return nil, err
	}
	// 	"x509KeyPem": "",
	gw.X509KeyPEM, err = GetString(data, "x509KeyPEM")
	if err != nil {
		return nil, err
	}
	// 	"apiListen": "",
	gw.ApiListen, err = GetString(data, "apiListen")
	if err != nil {
		return nil, err
	}
	// 	"apiX509CertPem": "",
	gw.ApiX509CertPEM, err = GetString(data, "apiX509CertPem")
	if err != nil {
		return nil, err
	}
	// 	"apiX509KeyPem": "",
	gw.ApiX509KeyPEM, err = GetString(data, "apiX509KeyPem")
	if err != nil {
		return nil, err
	}
	// 	"apiAccessToken": "",
	gw.ApiAccessToken, err = GetString(data, "apiAccessToken")
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
		_map, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.New(`"handler" data must be "map[string]interface{}" type`)
		}
		for k, v := range _map {
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
	Listen         string    // 监听地址
	X509CertPEM    string    // X509证书公钥，base64
	X509KeyPEM     string    // X509证书私钥，base64
	ApiListen      string    // 监听地址
	ApiX509CertPEM string    // X509证书公钥，base64
	ApiX509KeyPEM  string    // X509证书私钥，base64
	ApiAccessToken string    // 访问api的token
	interceptor    []Handler // 全局拦截
	notFound       []Handler // 匹配失败
	handler        sync.Map  // 处理函数路由表，string:[]Handler
}

// 开始服务
func (gw *Gateway) Serve() {
	// 检查
	if len(gw.interceptor) == 0 {
		gw.interceptor = []Handler{new(DefaultInterceptor)}
	}
	if len(gw.notFound) == 0 {
		gw.notFound = []Handler{new(DefaultNotFound)}
	}
	// 初始化服务
	server := &http.Server{Handler: gw}
	listener, err := net.Listen("tcp", gw.Listen)
	if err != nil {
		panic(err)
	}
	// 如果有证书，就使用tls
	if gw.X509CertPEM != "" && gw.X509KeyPEM != "" {
		certificate, err := tls.X509KeyPair([]byte(gw.X509CertPEM), []byte(gw.X509KeyPEM))
		if err != nil {
			panic(err)
		}
		listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{certificate},
		})
	}
	// 监听
	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
}

// 处理请求
func (gw *Gateway) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	ctx := contextPool.Get().(*Context)
	ctx.Req = req
	ctx.Res = res
	ctx.Data = nil
	// 拦截
	handler := gw.interceptor
	for _, h := range handler {
		if !h.Handle(ctx) {
			contextPool.Put(ctx)
			return
		}
	}
	// 获取handler
	ctx.Path = TopDir(req.URL.Path)
	value, ok := gw.handler.Load(ctx.Path)
	if !ok {
		handler = gw.notFound
		for _, h := range handler {
			if !h.Handle(ctx) {
				contextPool.Put(ctx)
				return
			}
		}
	}
	// 处理
	handler = value.([]Handler)
	for _, h := range handler {
		if !h.Handle(ctx) {
			contextPool.Put(ctx)
			return
		}
	}
	contextPool.Put(ctx)
}

// 设置处理，每个route对应一组handler
func (gw *Gateway) SetHandler(route string, handler ...Handler) {
	gw.handler.Store(TopDir(path.Clean("/"+route)), FilteNilHandler(handler...))
}

// 设置处理，每个route对应一组handler
func (gw *Gateway) SetInterceptor(handler ...Handler) {
	gw.interceptor = FilteNilHandler(handler...)
}

// 设置处理，每个route对应一组handler
func (gw *Gateway) SetNotFound(handler ...Handler) {
	gw.notFound = FilteNilHandler(handler...)
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
//		"handler3",
//		...
// ]
func (gw *Gateway) NewHandler(route string, data interface{}) error {
	handler, err := gw.newHandler("handler", data)
	if err != nil {
		return err
	}
	gw.SetHandler(route, handler...)
	return nil
}

// 创建并设置新的拦截，data的json格式
// [
// 		{
//			"name": "interceptor1",
//			...
//		},
// 		{
//			"name": "interceptor2",
//			...
//		},
//		"interceptor3",
//		...
// ]
func (gw *Gateway) NewInterceptor(data interface{}) error {
	handler, err := gw.newHandler("iterceptor", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.interceptor = handler
	}
	return nil
}

// 设置新的404，data的json格式
// [
// 		{
//			"name": "notfound1",
//			...
//		},
// 		{
//			"name": "notfound2",
//			...
//		},
//		"notfound3",
//		...
// ]
func (gw *Gateway) NewNotFound(data interface{}) error {
	handler, err := gw.newHandler("notFound", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.notFound = handler
	}
	return nil
}

// SetInterceptorBy和SetNotFoundBy的函数
func (gw *Gateway) newHandler(name string, data interface{}) ([]Handler, error) {
	array, ok := data.([]interface{})
	if !ok {
		return nil, fmt.Errorf(`"%s" data must be "[]interface{}" type`, name)
	}
	handler := make([]Handler, 0)
	for i, a := range array {
		hd, err := NewHandler(a)
		if err != nil {
			return nil, fmt.Errorf(`"%s" [%d] %s`, name, i, err.Error())
		}
		handler = append(handler, hd)
	}
	return handler, nil
}
