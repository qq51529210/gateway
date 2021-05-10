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

// 网关app
type Gateway struct {
	Listen      string    // 监听地址
	X509CertPEM string    // X509证书公钥，base64
	X509KeyPEM  string    // X509证书私钥，base64
	interceptor []Handler // 全局拦截
	notFound    []Handler // 匹配失败
	handler     sync.Map  // 处理函数路由表，string:[]Handler
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
	path := TopDir(req.URL.Path)
	value, ok := gw.handler.Load(path)
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

// 设置新的处理，data的json格式
// {
// 		"handler1": {
// 			...
// 		},
// 		"handler2": {
// 			...
// 		},
//		...
// },
func (gw *Gateway) SetHandlerBy(route string, data map[string]interface{}) error {
	for k, v := range data {
		handler, err := NewHandler(k, v)
		if err != nil {
			return fmt.Errorf(`"handler"."%s" %s`, k, err.Error())
		}
		gw.SetHandler(route, handler)
	}
	return nil
}

// 设置新的拦截，data的json格式
// {
// 		"name1": {
//			...
//		},
// 		"name2": {
//			...
//		}
// }
func (gw *Gateway) SetInterceptorBy(data map[string]interface{}) error {
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
// {
// 		"name1": {
//			...
//		},
// 		"name2": {
//			...
//		}
// }
func (gw *Gateway) SetNotFoundBy(data map[string]interface{}) error {
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
func (gw *Gateway) newHandler(name string, data map[string]interface{}) ([]Handler, error) {
	handler := make([]Handler, 0)
	for k, v := range data {
		hd, err := NewHandler(k, v)
		if err != nil {
			return nil, fmt.Errorf(`"%s"."%s" %s`, name, k, err.Error())
		}
		handler = append(handler, hd)
	}
	return handler, nil
}

// 创建Gateway实例，data的json格式
// {
// 	"listen": "",
// 	"certPem": "",
// 	"keyPem": "",
// 	"interceptor": {
// 		"name1": {
//			...
//		},
// 		"name2": {
//			...
//		},
//		...
// 	},
// 	"notfound": {
// 		"name1": {
//			...
//		},
// 		"name2": {
//			...
//		},
//		...
// 	},
// 	"handler": {
// 		"/path1": {
// 			"handler1": {
// 				...
// 			},
// 			"handler2": {
// 				...
// 			},
//			...
// 		},
// 		"/path2": {
// 			"handler3": {
// 				...
// 			},
//			...
// 		},
// }
func NewGateway(data map[string]interface{}) (*Gateway, error) {
	var err error
	gw := new(Gateway)
	// Listen
	gw.Listen, err = MustGetString(data, "listen")
	if err != nil {
		return nil, err
	}
	// X509CertPEM
	gw.X509CertPEM, err = GetString(data, "x509CertPEM")
	if err != nil {
		return nil, err
	}
	// X509KeyPEM
	gw.X509KeyPEM, err = GetString(data, "x509KeyPEM")
	if err != nil {
		return nil, err
	}
	// 拦截器
	value, ok := data["interceptor"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.New(`"interceptor" value must be "map[string]interface{}"`)
		}
		err = gw.SetInterceptorBy(m)
		if err != nil {
			return nil, err
		}
	}
	// 404
	value, ok = data["notfound"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.New(`"notfound" value must be "map[string]interface{}"`)
		}
		err = gw.SetNotFoundBy(m)
		if err != nil {
			return nil, err
		}
	}
	// 处理器
	value, ok = data["handler"]
	if ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, errors.New(`"handler" value must be "map[string]interface{}"`)
		}
		// "/path1": {...}
		// "/path2": {...}
		for k, v := range m {
			d, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf(`"handler"."%s" value must be "map[string]interface{}"`, k)
			}
			err = gw.SetHandlerBy(k, d)
			if err != nil {
				return nil, err
			}
		}
	}
	return gw, nil
}
