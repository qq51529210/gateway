package handler

import (
	"mime"
	"net/http"
	"testing"
)

func Test_IPInterceptor(t *testing.T) {
	var data IPInterceptorData
	data.StatusCode = 201
	data.ContentType = mime.TypeByExtension(".json")
	data.Message = `{"message": "Accept denied!"}`
	h, err := NewHandler(IPInterceptorRegisterName(), &data)
	if err != nil {
		t.Fatal(err)
	}

	ip := "192.168.1.2"
	ipInterceptor := h.(*IPInterceptor)
	ipInterceptor.redis.Cmd("del", ip)
	ipInterceptor.redis.Cmd("set", ip, 1)

	res := &testResponse{}
	var c Context
	c.Req = &http.Request{
		RemoteAddr: ip + ":12345",
	}
	c.Res = res
	if h.Handle(&c) {
		t.FailNow()
	}

	if res.statusCode != data.StatusCode {
		t.FailNow()
	}
	if res.Header().Get("Content-Type") != data.ContentType {
		t.FailNow()
	}
	if res.body.String() != data.Message {
		t.FailNow()
	}

	ip = "192.168.1.3"
	res.Reset()
	c.Req.RemoteAddr = ip + ":12345"
	if !h.Handle(&c) {
		t.FailNow()
	}
	if res.statusCode == data.StatusCode {
		t.FailNow()
	}
	if res.Header().Get("Content-Type") == data.ContentType {
		t.FailNow()
	}
	if res.body.String() == data.Message {
		t.FailNow()
	}
}
