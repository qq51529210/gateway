package main

import (
	"testing"

	"github.com/qq51529210/gateway/handler"
	"github.com/qq51529210/redis"
)

func Test_NewGateway(t *testing.T) {
	gw, err := NewGateway(&NewGatewayData{
		Listen: ":3390",
		NotFound: []*NewHandlerData{
			{
				Name: handler.DefaultNotFoundRegisterName(),
			},
		},
		Intercept: []*NewHandlerData{
			{
				Name: handler.IPInterceptorRegisterName(),
				Data: &handler.IPInterceptorData{
					Redis: &redis.ClientConfig{},
				},
			},
			{
				Name: handler.AuthenticationInterceptorRegisterName(),
				Data: &handler.AuthenticationInterceptorData{
					CookieName: "token",
					Redis:      &redis.ClientConfig{},
				},
			},
		},
		Forward: map[string][]*NewHandlerData{
			"service1": {
				{
					Name: handler.DefaultForwarderName(),
					Data: &handler.NewDefaultForwarderData{
						RequestUrl: "http://127.0.0.1:3391",
					},
				},
			},
			"service2": {
				{
					Name: handler.DefaultForwarderName(),
					Data: &handler.NewDefaultForwarderData{
						RequestUrl: "http://127.0.0.1:3391",
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	gw.Close()
}
