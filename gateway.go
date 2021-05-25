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

type NewHandlerData struct {
	// Use for create handler.
	Name string `json:"name"`
	// Handler init data.
	Data interface{} `json:"data"`
}

type NewGatewayData struct {
	// Gateway server listen address.
	// If X509CertPEM and X509KeyPEM both are not empty,gateway server will use TLS.
	Listen string `json:"listen"`
	// Gateway server x509 cert file data.
	X509CertPEM string `json:"x509CertPEM"`
	// Gateway server x509 key file data.
	X509KeyPEM string `json:"x509KeyPEM"`
	// Gateway interceptor handler call chain.
	Intercept []*NewHandlerData `json:"intercept"`
	// Gateway notFound handler call chain.
	NotFound []*NewHandlerData `json:"notFound"`
	// Gateway forward handler call chain.
	Forward map[string][]*NewHandlerData `json:"forward"`
	// API management server listen address.
	// If ApiX509CertPEM and ApiX509KeyPEM both are not empty,api server will use TLS.
	ApiListen string `json:"apiListen"`
	// API management server x509 cert file data.
	ApiX509CertPEM string `json:"apiX509CertPEM"`
	// API management server x509 key file data.
	ApiX509KeyPEM string `json:"apiX509KeyPEM"`
	// API management server authentication token.
	ApiAccessToken string `json:"apiAccessToken"`
}

type gatewayForwarder struct {
	RegisterName string
	handler.Handler
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
	// Init intercept handler call chain.
	err = gw.newIntercept(data.Intercept)
	if err != nil {
		return nil, err
	}
	// Init notfound handler call chain.
	err = gw.newNotFound(data.NotFound)
	if err != nil {
		return nil, err
	}
	// Init forward handler call chain.
	for k, v := range data.Forward {
		err = gw.newForward(k, v)
		if err != nil {
			return nil, err
		}
	}
	return gw, nil
}

type Gateway struct {
	// Gateway server.
	server http.Server
	// Gateway server listener.
	listener net.Listener
	// Api server.
	apiServer http.Server
	// Api server listener.
	apiListener net.Listener
	// Api server token
	apiToken string
	// Gateway intercept chain.
	intercept []handler.Handler
	// Gateway notfound chain.
	notfound []handler.Handler
	// Gateway forward chain,key is route and value is *gatewayForwarder.
	forward sync.Map
}

func (gw *Gateway) Serve() error {
	// Default interceptor handler.
	if len(gw.intercept) == 0 {
		gw.intercept = []handler.Handler{new(handler.DefaultInterceptor)}
	}
	// Default notFound handler.
	if len(gw.notfound) == 0 {
		gw.notfound = []handler.Handler{new(handler.DefaultNotFound)}
	}
	// Set handler function.
	gw.server.Handler = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		ctx := contextPool.Get().(*handler.Context)
		ctx.Req = req
		ctx.Res = res
		ctx.Data = nil
		// Intercept  chain.
		for _, h := range gw.intercept {
			if !h.Handle(ctx) {
				contextPool.Put(ctx)
				return
			}
		}
		ctx.Path = handler.TopDir(req.URL.Path)
		value, ok := gw.forward.Load(ctx.Path)
		if !ok {
			// NotFound chain.
			for _, h := range gw.notfound {
				if !h.Handle(ctx) {
					break
				}
			}
		} else {
			// Forward chain.
			ctx.Forward.Header = make(http.Header)
			for _, h := range value.([]handler.Handler) {
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

// Close gateway server and api server.
func (gw *Gateway) Close() error {
	gw.server.Close()
	gw.apiServer.Close()
	return nil
}

// Setup forwarder chain.
func (gw *Gateway) newForward(route string, data []*NewHandlerData) error {
	route = handler.TopDir(route)
	if route == "" {
		return errors.New(`"forward"."route" must define`)
	}
	if route[0] != '/' {
		route = "/" + route
	}
	if len(data) == 0 {
		return errors.New(`"forward"."route[%s]" must define handler`)
	}
	forward := make([]*gatewayForwarder, 0)
	for i, a := range data {
		hd, err := handler.NewHandler(a.Name, a.Data)
		if err != nil {
			return fmt.Errorf(`"forward[%d]" %s`, i, err.Error())
		}
		forward = append(forward, &gatewayForwarder{
			RegisterName: a.Name,
			Handler:      hd,
		})
	}
	value, ok := gw.forward.LoadOrStore(route, forward)
	if ok {
		for _, h := range value.([]*gatewayForwarder) {
			h.Release()
		}
	}
	return nil
}

// Setup iterceptor chain.
func (gw *Gateway) newIntercept(data []*NewHandlerData) error {
	if len(data) == 0 {
		return errors.New(`"itercept" must define handler`)
	}
	for _, h := range gw.intercept {
		h.Release()
	}
	gw.intercept = make([]handler.Handler, 0)
	for i, a := range data {
		hd, err := handler.NewHandler(a.Name, a.Data)
		if err != nil {
			return fmt.Errorf(`"itercept[%d]" %s`, i, err.Error())
		}
		gw.intercept = append(gw.intercept, hd)
	}
	return nil
}

// Setup notfound chain.
func (gw *Gateway) newNotFound(data []*NewHandlerData) error {
	if len(data) == 0 {
		return errors.New(`"notfound" must define handler`)
	}
	for _, h := range gw.notfound {
		h.Release()
	}
	gw.notfound = make([]handler.Handler, 0)
	for i, a := range data {
		hd, err := handler.NewHandler(a.Name, a.Data)
		if err != nil {
			return fmt.Errorf(`"notfound[%d]" %s`, i, err.Error())
		}
		gw.notfound = append(gw.notfound, hd)
	}
	return nil
}

// Api management serve.
func (gw *Gateway) ApiServe() error {
	if gw.apiListener == nil {
		return errors.New("api server didn't listen")
	}
	// Setup api router.
	var rr router.MethodRouter
	rr.Interceptor = []router.HandleFunc{func(c *router.Context) bool {
		// Check authentication token.
		if c.BearerToken() != gw.apiToken {
			c.Response.WriteHeader(http.StatusFound)
			return false
		}
		return true
	}}
	rr.AddPut("/api/intercepts", gw.apiPutIntercept)
	rr.AddPut("/api/notfounds", gw.apiPutNotFound)
	rr.AddPut("/api/forwards", gw.apiPutForward)
	rr.AddPut("/api/token", gw.apiPutToken)
	// Start serve
	gw.server.Handler = &rr
	// Start serve
	return gw.server.Serve(gw.listener)
}

// Put new intercept chain.
func (gw *Gateway) apiPutIntercept(c *router.Context) bool {
	return true
}

// Put new notfound chain.
func (gw *Gateway) apiPutNotFound(c *router.Context) bool {
	return true
}

// Put new forward chain.
func (gw *Gateway) apiPutForward(c *router.Context) bool {
	return true
}

// Put new token.
func (gw *Gateway) apiPutToken(c *router.Context) bool {
	return true
}
