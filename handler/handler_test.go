package handler

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"testing"
)

type testResponse struct {
	statusCode int
	header     http.Header
	body       strings.Builder
}

func (r *testResponse) Header() (h http.Header) {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *testResponse) Write(b []byte) (n int, err error) {
	return r.body.Write(b)
}

func (r *testResponse) WriteString(s string) (n int, err error) {
	return r.body.WriteString(s)
}

func (r *testResponse) WriteHeader(n int) {
	r.statusCode = n
}

func (r *testResponse) Reset() {
	r.statusCode = 0
	r.header = make(http.Header)
	r.body.Reset()
}

func Test_DefaultForwarder(t *testing.T) {
	d := &NewDefaultForwarderData{
		RequestUrl:             "http://127.0.0.1:3391",
		RequestTimeout:         3000,
		RequestHeader:          []string{"Req-Header1", "Req-Header2", "Req-Header3"},
		RequestAdditionHeader:  map[string]string{"Req-Add-Header1": "1", "Req-Add-Header2": "2"},
		ResponseAdditionHeader: map[string]string{"Res-Add-Header1": "1", "Res-Add-Header2": "2"},
	}
	h, err := NewHandler(DefaultForwarderName(), d)
	if err != nil {
		t.Fatal(err)
	}
	reqBody := "req body"
	resCode := 201
	resHeader := []string{"Res-Header1", "Res-Header2"}
	resBody := "res body"
	// Service serve
	var ser http.Server
	var serErr error
	go func() {
		ser.Addr = "127.0.0.1:3391"
		ser.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Check headers.
			for _, k := range d.RequestHeader {
				_, o := r.Header[k]
				if !o {
					serErr = fmt.Errorf("header %s not found", k)
					return
				}
			}
			// Check addition headers.
			for k := range d.RequestAdditionHeader {
				_, o := r.Header[k]
				if !o {
					serErr = fmt.Errorf("addition header %s not found", k)
					return
				}
			}
			// Response
			for _, k := range resHeader {
				rw.Header().Add(k, "1")
			}
			rw.WriteHeader(resCode)
			io.WriteString(rw, resBody)
		})
		ser.ListenAndServe()
	}()
	// Forward handle.
	var c Context
	var body bytes.Buffer
	body.WriteString(reqBody)
	c.Req, err = http.NewRequest(http.MethodGet, "http://127.0.0.1:3390/service1", &body)
	if err != nil {
		t.Fatal(err)
	}
	// Request headers.
	for _, s := range d.RequestHeader {
		c.Req.Header.Add(s, "1")
	}
	res := new(testResponse)
	c.Res = res
	if h.Handle(&c) {
		t.FailNow()
	}
	ser.Close()
	// Check server error.
	if serErr != nil {
		t.Fatal(serErr)
	}
	// Check status code.
	if res.statusCode != resCode {
		t.FailNow()
	}
	// Check headers.
	for _, k := range resHeader {
		_, o := res.header[k]
		if !o {
			t.FailNow()
		}
	}
	// Check addition headers.
	for k := range d.ResponseAdditionHeader {
		_, o := res.header[k]
		if !o {
			t.FailNow()
		}
	}
	// Check body.
	if res.body.String() != resBody {
		t.FailNow()
	}
}

func Test_DefaultNotFound(t *testing.T) {
	var data InterceptData
	data.StatusCode = 401
	data.ContentType = mime.TypeByExtension(".json")
	data.Message = `{"message": "Service not found!"}`
	h, err := NewHandler(DefaultNotFoundRegisterName(), &data)
	if err != nil {
		t.Fatal(err)
	}

	res := &testResponse{}
	var c Context
	c.Res = res
	if h.Handle(&c) || res.statusCode != data.StatusCode || res.Header().Get("Content-Type") != data.ContentType || res.body.String() != data.Message {
		t.FailNow()
	}
}
