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

type NewGatewayData struct {
	Listen         string                               // 监听地址，必须
	X509CertPEM    string                               // X509证书公钥，base64，可选
	X509KeyPEM     string                               // X509证书私钥，base64，可选
	ApiListen      string                               // api服务监听地址，可选
	ApiX509CertPEM string                               // api服务X509证书公钥，base64，可选
	ApiX509KeyPEM  string                               // api服务X509证书私钥，base64，可选
	ApiAccessToken string                               // api服务访问token，可选
	Interceptor    []*handler.NewHandlerData            // 生成Handler参数，可选
	NotFound       []*handler.NewHandlerData            // 生成Handler参数，可选
	Handler        map[string][]*handler.NewHandlerData // 生成Handler参数，可选
}

// 创建Gateway实例
func NewGateway(data *NewGatewayData) (*Gateway, error) {
	var err error
	gw := new(Gateway)
	gw.Listen = data.Listen
	if gw.Listen == "" {
		return nil, errors.New(`"listen" data must not be empty`)
	}
	gw.X509CertPEM = data.X509CertPEM
	gw.X509KeyPEM = data.X509KeyPEM
	gw.ApiListen = data.ApiListen
	gw.ApiX509CertPEM = data.ApiX509CertPEM
	gw.ApiX509KeyPEM = data.ApiX509KeyPEM
	gw.ApiAccessToken = data.ApiAccessToken
	err = gw.NewInterceptor(data.Interceptor)
	if err != nil {
		return nil, fmt.Errorf(`"interceptor" %s`, err.Error())
	}
	err = gw.NewNotFound(data.NotFound)
	if err != nil {
		return nil, fmt.Errorf(`"interceptor" %s`, err.Error())
	}
	for k, v := range data.Handler {
		err = gw.NewHandler(k, v)
		if err != nil {
			return nil, fmt.Errorf(`"handler"."%s" %s`, k, err.Error())
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
	} else {
		// 处理
		ctx.Handler = value.([]handler.Handler)
		for _, h := range ctx.Handler {
			if !h.Handle(ctx) {
				break
			}
		}
	}
	contextPool.Put(ctx)
}

// 设置处理，每个route对应一组handler
func (gw *Gateway) SetHandler(route string, handler ...handler.Handler) {
	gw.handler.Store(util.TopDir(path.Clean("/"+route)), handler)
}

// 创建并设置新的处理，如果没有生成新的Handler，则不生效。
func (gw *Gateway) NewHandler(route string, data []*handler.NewHandlerData) error {
	handler, err := newHandler("handler", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.SetHandler(route, handler...)
	}
	return nil
}

// 创建并设置新的interceptor，如果没有生成新的Handler，则不生效。
func (gw *Gateway) NewInterceptor(data []*handler.NewHandlerData) error {
	handler, err := newHandler("iterceptor", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.Interceptor = handler
	}
	return nil
}

// 设置新的notfound，如果没有生成新的Handler，则不生效。
func (gw *Gateway) NewNotFound(data []*handler.NewHandlerData) error {
	handler, err := newHandler("notFound", data)
	if err != nil {
		return err
	}
	if len(handler) > 0 {
		gw.NotFound = handler
	}
	return nil
}

func newHandler(name string, data []*handler.NewHandlerData) ([]handler.Handler, error) {
	newHD := make([]handler.Handler, 0)
	// data = util.SortHandlerData(data)
	for i, a := range data {
		hd, err := handler.NewHandler(a)
		if err != nil {
			return nil, fmt.Errorf(`"%s[%d]" %s`, name, i, err.Error())
		}
		newHD = append(newHD, hd)
	}
	return newHD, nil
}
