package handler

import (
	"mime"
	"net/http"
	"testing"
)

func Test_AuthenticationInterceptor(t *testing.T) {
	var data AuthenticationInterceptorData
	data.StatusCode = 202
	data.ContentType = mime.TypeByExtension(".json")
	data.Message = `{"message": "Unauthorized access!"}`
	data.CookieName = "session_id"
	h, err := NewHandler(AuthenticationInterceptorRegisterName(), &data)
	if err != nil {
		t.Fatal(err)
	}

	token := "token1"
	hd := h.(*AuthenticationInterceptor)
	hd.redis.Cmd("del", token)
	hd.redis.Cmd("set", token, 1)

	res := &testResponse{}
	var c Context
	c.Res = res
	c.Req = &http.Request{}
	c.Req.Header = make(http.Header)
	c.Req.Header.Add("Authorization", "Bearer "+token)
	if !h.Handle(&c) || res.statusCode == data.StatusCode || res.Header().Get("Content-Type") == data.ContentType || res.body.String() == data.Message {
		t.FailNow()
	}

	c.Req.Header = make(http.Header)
	c.Req.AddCookie(&http.Cookie{Name: data.CookieName, Value: token})
	if !h.Handle(&c) || res.statusCode == data.StatusCode || res.Header().Get("Content-Type") == data.ContentType || res.body.String() == data.Message {
		t.FailNow()
	}

	token = "token2"
	res.Reset()
	c.Req.Header = make(http.Header)
	c.Req.AddCookie(&http.Cookie{Name: data.CookieName, Value: token})
	c.Req.Header.Add("Authorization", "Bearer "+token)
	if h.Handle(&c) || res.statusCode != data.StatusCode || res.Header().Get("Content-Type") != data.ContentType || res.body.String() != data.Message {
		t.FailNow()
	}
}
