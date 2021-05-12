package gateway

import "testing"

func Test_TopDir(t *testing.T) {
	s := TopDir("/a")
	if s != "/a" {
		t.FailNow()
	}
	s = TopDir("a/b")
	if s != "a" {
		t.FailNow()
	}
}

func Test_HandlerName(t *testing.T) {
	s := HandlerName(new(DefaultHandler))
	if s != "github.com/qq51529210/gateway.DefaultHandler" {
		t.FailNow()
	}
	s = HandlerName(new(DefaultInterceptor))
	if s != "github.com/qq51529210/gateway.DefaultInterceptor" {
		t.FailNow()
	}
	s = HandlerName(new(DefaultNotFound))
	if s != "github.com/qq51529210/gateway.DefaultNotFound" {
		t.FailNow()
	}
}

func Test_MustGetString(t *testing.T) {
	m := make(map[string]interface{})
	m["a"] = 1
	m["b"] = "1"
	s, err := MustGetString(m, "a")
	if err == nil || s != "" {
		t.FailNow()
	}
	s, err = MustGetString(m, "b")
	if err != nil || s != "1" {
		t.FailNow()
	}
	s, err = MustGetString(m, "c")
	if err == nil || s != "" {
		t.FailNow()
	}
}

func Test_GetString(t *testing.T) {
	m := make(map[string]interface{})
	m["a"] = 1
	m["b"] = "1"
	s, err := GetString(m, "a")
	if err == nil || s != "" {
		t.FailNow()
	}
	s, err = GetString(m, "b")
	if err != nil || s != "1" {
		t.FailNow()
	}
	s, err = GetString(m, "c")
	if err != nil || s != "" {
		t.FailNow()
	}
}

func Test_FilteNilHandler(t *testing.T) {
	hd := FilteNilHandler(nil)
	if len(hd) != 0 {
		t.FailNow()
	}
	h1 := &DefaultHandler{}
	h2 := &DefaultInterceptor{}
	hd = append(hd, h1)
	hd = append(hd, nil)
	hd = append(hd, h2)
	hd = FilteNilHandler(hd...)
	if len(hd) != 2 || hd[0] != h1 || hd[1] != h2 {
		t.FailNow()
	}
}
