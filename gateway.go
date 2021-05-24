package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/qq51529210/gateway/handler"
	router "github.com/qq51529210/http-router"
)

var (
	contextPool = new(sync.Pool)
)

func init() {
	contextPool.New = func() interface{} {
		return new(handler.Context)
	}
}

// Create handler chain by data.
func newHandlerChain(name string, data []*handler.NewHandlerData) ([]handler.Handler, error) {
	newHD := make([]handler.Handler, 0)
	for i, a := range data {
		hd, err := handler.NewHandler(a)
		if err != nil {
			return nil, fmt.Errorf(`"%s[%d]" %s`, name, i, err.Error())
		}
		newHD = append(newHD, hd)
	}
	return newHD, nil
}

type NewGatewayData struct {
	// Gateway server listen address.
	Listen string `json:"listen"`
	// Gateway server x509 cert file data.
	X509CertPEM string `json:"x509CertPEM"`
	// Gateway server x509 key file data.
	X509KeyPEM string `json:"x509KeyPEM"`
	// Gateway interceptor handler call chain.
	Intercept []*handler.NewHandlerData `json:"intercept"`
	// Gateway notFound handler call chain.
	NotFound []*handler.NewHandlerData `json:"notFound"`
	// Gateway forward handler call chain.
	Forward map[string][]*handler.NewHandlerData `json:"forward"`
	// API management server listen address.
	ApiListen string `json:"apiListen"`
	// API management server x509 cert file data.
	ApiX509CertPEM string `json:"apiX509CertPEM"`
	// API management server x509 key file data.
	ApiX509KeyPEM string `json:"apiX509KeyPEM"`
	// API management server authentication token.
	ApiAccessToken string `json:"apiAccessToken"`
}

// Create a new Gateway
func NewGateway(data *NewGatewayData) (*Gateway, error) {
	var err error
	gw := new(Gateway)
	// Create listener
	if data.Listen == "" {
		return nil, errors.New(`"listen" is empty`)
	}
	listener, err := net.Listen("tcp", data.Listen)
	if err != nil {
		return nil, err
	}
	// Gateway http server use TLS?
	if data.X509CertPEM != "" && data.X509KeyPEM != "" {
		certificate, err := tls.X509KeyPair([]byte(data.X509CertPEM), []byte(data.X509KeyPEM))
		if err != nil {
			return nil, err
		}
		gw.listener = tls.NewListener(listener, &tls.Config{
			Certificates: []tls.Certificate{certificate},
		})
	}
	// Api http server use TLS?
	if data.ApiListen != "" {
		listener, err = net.Listen("tcp", data.ApiListen)
		if err != nil {
			return nil, err
		}
		if data.ApiX509CertPEM != "" && data.ApiX509KeyPEM != "" {
			certificate, err := tls.X509KeyPair([]byte(data.ApiX509CertPEM), []byte(data.ApiX509KeyPEM))
			if err != nil {
				return nil, err
			}
			gw.apiListener = tls.NewListener(listener, &tls.Config{
				Certificates: []tls.Certificate{certificate},
			})
		}
	}
	// Init interceptor call chain.
	err = gw.newInterceptor(data.Intercept)
	if err != nil {
		return nil, fmt.Errorf(`"interceptor" %s`, err.Error())
	}
	// Init notfound call chain.
	err = gw.newNotFound(data.NotFound)
	if err != nil {
		return nil, fmt.Errorf(`"notfound" %s`, err.Error())
	}
	// Init forward call chain.
	for k, v := range data.Forward {
		err = gw.newForwarder(k, v)
		if err != nil {
			return nil, fmt.Errorf(`"handler"."%s" %s`, k, err.Error())
		}
	}
	return gw, nil
}

type Gateway struct {
	// Gateway http server.
	server http.Server
	// Gateway http server listener.
	listener net.Listener
	// Api http server.
	apiServer http.Server
	// Api http server listener.
	apiListener net.Listener
	// Gateway interceptor call chain.
	intercept []handler.Handler
	// Gateway notfound call chain.
	notFound []handler.Handler
	// Gateway forward call chain.
	// Key is route and value is call chain.
	forward sync.Map
}

func (gw *Gateway) Serve() error {
	// Default interceptor handler.
	if len(gw.intercept) == 0 {
		gw.intercept = []handler.Handler{new(handler.DefaultInterceptor)}
	}
	// Default notFound handler.
	if len(gw.notFound) == 0 {
		gw.notFound = []handler.Handler{new(handler.DefaultNotFound)}
	}
	// Set handler function.
	gw.server.Handler = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := contextPool.Get().(*handler.Context)
		ctx.Req = req
		ctx.Res = res
		ctx.Data = nil
		// Interceptor call chain.
		ctx.Interceptor = gw.intercept
		for _, h := range ctx.Interceptor {
			if !h.Handle(ctx) {
				contextPool.Put(ctx)
				return
			}
		}
		ctx.Path = handler.TopDir(req.URL.Path)
		value, ok := gw.forward.Load(ctx.Path)
		if !ok {
			// NotFound call chain.
			ctx.NotFound = gw.notFound
			for _, h := range ctx.NotFound {
				if !h.Handle(ctx) {
					break
				}
			}
		} else {
			// NotFound call chain.
			ctx.Forward = value.([]handler.Handler)
			for _, h := range ctx.Forward {
				if !h.Handle(ctx) {
					break
				}
			}
		}
		contextPool.Put(ctx)
	})
	// Start serve
	return gw.server.Serve(gw.listener)
}

func (gw *Gateway) ApiServe() error {
	if gw.apiListener == nil {
		return errors.New("api server didn't listen")
	}
	// Setup api router.
	var router router.MethodRouter
	router.AddPut("/api/interceptors", gw.apiPutInterceptor)
	// Start serve
	gw.server.Handler = &router
	// Start serve
	return gw.server.Serve(gw.listener)
}

// Close gateway server and api server.
func (gw *Gateway) Close() error {
	gw.server.Close()
	gw.apiServer.Close()
	return nil
}

// Setup forwarder call chain.
func (gw *Gateway) newForwarder(route string, data []*handler.NewHandlerData) error {
	route = handler.TopDir(route)
	if route == "" {
		return errors.New(`"route" is empty`)
	}
	if len(data) == 0 {
		return errors.New(`"handler" is empty`)
	}
	if route[0] != '/' {
		route = "/" + route
	}
	hd, err := newHandlerChain("handler", data)
	if err != nil {
		return err
	}
	gw.forward.Store(route, hd)
	return nil
}

// Setup iterceptor call chain.
func (gw *Gateway) newInterceptor(data []*handler.NewHandlerData) error {
	if len(data) == 0 {
		return errors.New(`"handler" is empty`)
	}
	hd, err := newHandlerChain("iterceptor", data)
	if err != nil {
		return err
	}
	if len(hd) > 0 {
		gw.intercept = hd
	}
	return nil
}

// Setup notfound call chain.
func (gw *Gateway) newNotFound(data []*handler.NewHandlerData) error {
	if len(data) == 0 {
		return errors.New(`"handler" is empty`)
	}
	hd, err := newHandlerChain("notfound", data)
	if err != nil {
		return err
	}
	gw.notFound = hd
	return nil
}

func (gw *Gateway) apiPutInterceptor(c *router.Context) bool {
	return true
}
