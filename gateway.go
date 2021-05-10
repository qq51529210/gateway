package gateway

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"sync"
)

// 网关app
type Gateway struct {
	Listen      string           // 监听地址
	X509CertPEM string           // X509证书公钥，base64
	X509KeyPEM  string           // X509证书私钥，base64
	Interceptor Interceptor      // 拦截函数
	NotFound    http.HandlerFunc // 匹配失败函数
	handler     sync.Map         // 转发接口，key:url.Path，value:http.Handler
}

// 开始服务
func (gw *Gateway) Serve() {
	// 检查
	if gw.Interceptor == nil {
		gw.Interceptor = new(DefaultInterceptor)
	}
	if gw.NotFound == nil {
		gw.NotFound = func(res http.ResponseWriter, req *http.Request) {
			res.WriteHeader(http.StatusNotFound)
		}
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
	// 拦截
	if gw.Interceptor.Intercept(res, req) {
		return
	}
	// 获取handler
	path := topDir(req.URL.Path)
	handler, ok := gw.handler.Load(path)
	if !ok {
		gw.NotFound(res, req)
		return
	}
	// 处理
	handler.(http.Handler).ServeHTTP(res, req)
}

// 设置路由的handler，每个代理都可以有不同的handler
func (gw *Gateway) SetHandler(route string, handler http.Handler) error {
	gw.handler.Store(topDir(path.Clean("/"+route)), handler)
	return nil
}

// 找出第一层目录，/a/b中的/a
func topDir(path string) string {
	for i := 1; i < len(path); i++ {
		if path[i] == '/' {
			path = path[:i]
		}
	}
	return path
}

// data必须有key为name，value为string类型的数据
func getString(data map[string]interface{}, name string) (string, error) {
	val, ok := data[name]
	if !ok {
		return "", fmt.Errorf(`"%s" must be define`, name)
	}
	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf(`"%s" must be string`, name)
	}
	return str, nil
}

// data如果有key为name的数据，那么value必须string类型
func hasString(data map[string]interface{}, name string) (string, error) {
	val, ok := data[name]
	if ok {
		str, ok := val.(string)
		if !ok {
			return "", fmt.Errorf(`"%s" must be string`, name)
		}
		return str, nil
	}
	return "", nil
}

// 根据data生成具体的Gateway。
// data数据结构如下
// {
// 	"listen": "",
// 	"certPem": "",
// 	"keyPem": "",
// 	"interceptor": {
// 		...
// 	},
// 	"handler": {
// 		"routePath1":{
// 			"type":"",
// 			...
// 		},
// 		"routePath2":{
// 			"type":"",
// 			...
// 		}
// 	}
// }
func NewGateway(data map[string]interface{}) (*Gateway, error) {
	var gw Gateway
	var err error
	// Listen
	gw.Listen, err = getString(data, "listen")
	if err != nil {
		return nil, err
	}
	// X509CertPEM
	gw.X509CertPEM, err = hasString(data, "x509CertPEM")
	if err != nil {
		return nil, err
	}
	// X509KeyPEM
	gw.X509KeyPEM, err = hasString(data, "x509KeyPEM")
	if err != nil {
		return nil, err
	}
	// 拦截器
	val, ok := data["interceptor"]
	if !ok {
		gw.Interceptor, err = NewInterceptor(nil)
	} else {
		gw.Interceptor, err = NewInterceptor(val.(map[string]interface{}))
	}
	if err != nil {
		return nil, fmt.Errorf(`"interceptor" %s`, err.Error())
	}
	// 处理器
	val, ok = data["handler"]
	if ok {
		m, ok := val.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf(`"handler" value must be "map[string]interface{}"`)
		}
		for k, v := range m {
			d, ok := v.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf(`"handler"."%s" value must be "map[string]interface{}"`, k)
			}
			handler, err := NewHandler(d)
			if err != nil {
				return nil, fmt.Errorf(`"handler"."%s" %s`, k, err.Error())
			}
			gw.SetHandler(k, handler)
		}
	}
	return &gw, nil
}
