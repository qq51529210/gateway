package handler

import (
	"bytes"
	"fmt"
	"io"
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

func Test_DefaultForwarder(t *testing.T) {
	d := &NewDefaultForwarderData{
		RequestUrl:             "http://127.0.0.1:3391",
		RequestTimeout:         3000,
		RequestHeader:          []string{"reqHeader1", "reqHeader2", "reqHeader3"},
		RequestAdditionHeader:  map[string]string{"reqAddHeader1": "1", "reqAddHeader2": "2"},
		ResponseAdditionHeader: map[string]string{"resAddHeader1": "1", "resAddHeader2": "2"},
	}
	h, err := NewHandler(DefaultForwarderName(), d)
	if err != nil {
		t.Fatal(err)
	}
	reqBody := "req body"
	resCode := 201
	resHeader := []string{"resHeader1", "resHeader2"}
	resBody := "res body"
	var ser http.Server
	var serErr error
	go func() {
		ser.Addr = "127.0.0.1:3391"
		ser.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			defer ser.Close()
			// Check headers.
			for _, k := range d.RequestHeader {
				_, o := r.Header[k]
				if !o {
					serErr = fmt.Errorf("header %s not found", k)
					return
				}
			}
			// Check addition headers.
			for _, k := range d.RequestAdditionHeader {
				_, o := r.Header[k]
				if !o {
					serErr = fmt.Errorf("addition header %s not found", k)
					return
				}
			}
			// Response
			rw.WriteHeader(resCode)
			for _, k := range resHeader {
				rw.Header().Add(k, "1")
			}
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
	h.Handle(&c)
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
