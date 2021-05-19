package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/qq51529210/gateway/handler"
)

type testHandler struct {
	data string
}

func (h *testHandler) Handle(c *handler.Context) bool {
	return true
}

func (h *testHandler) Update(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf(`data must be "map[string]interface{}" type`)
	}
	data, ok = m["data"]
	if !ok {
		return fmt.Errorf(`"data" must be defined`)
	}
	h.data = data.(string)
	return nil
}

func (h *testHandler) Name() string {
	return "testHandler"
}

type testService struct {
	http.Server
	path              string            // 验证请求的路径
	reqHeader         map[string]string // 验证请求的header
	reqAdditionHeader map[string]string // 验证请求的header
	reqBody           string            // 验证请求的body
	resCode           int               // 响应的status code
	resHeader         map[string]string // 响应的header
	resBody           string            // 响应的body
	err               error             // 保存错误信息
}

func (ts *testService) Serve(address, path, reqBody, resBody string, reqHeader, reqAdditionHeader, resHeader map[string]string, resCode int) {
	ts.reqHeader = reqHeader
	ts.reqAdditionHeader = reqAdditionHeader
	ts.reqBody = reqBody
	ts.resHeader = resHeader
	ts.resBody = resBody
	ts.path = path
	ts.resCode = resCode

	ts.Addr = address
	ts.Handler = ts
	ts.ListenAndServe()
}

func (ts *testService) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	// 验证请求路径
	if req.URL.Path+"?"+req.URL.RawQuery != ts.path {
		ts.err = fmt.Errorf(`"request path" is "%s" no "%s"`, req.URL.Path, ts.path)
		return
	}
	// 验证请求的header
	for k := range ts.reqHeader {
		if ts.reqHeader[k] != req.Header.Get(k) {
			ts.err = fmt.Errorf(`"request header"."%s" is "%s" no "%s"`, k, req.Header.Get(k), ts.reqHeader[k])
			return
		}
	}
	for k := range ts.reqAdditionHeader {
		if ts.reqAdditionHeader[k] != req.Header.Get(k) {
			ts.err = fmt.Errorf(`"request addition header"."%s" is "%s" no "%s"`, k, req.Header.Get(k), ts.reqAdditionHeader[k])
			return
		}
	}
	// 验证请求的body
	var body strings.Builder
	_, ts.err = io.Copy(&body, req.Body)
	if ts.err != nil {
		return
	}
	if body.String() != ts.reqBody {
		ts.err = fmt.Errorf(`"request body" is "%s" no "%s"`, body.String(), ts.reqBody)
		return
	}
	// 响应
	res.WriteHeader(ts.resCode)
	for k, v := range ts.resHeader {
		res.Header().Add(k, v)
	}
	_, ts.err = io.WriteString(res, ts.resBody)
}

// url表示请求的地址，reqHeader表示请求的header，reqBody表示请求的body，
// resCode表示验证响应的status code，resHeader表示验证响应的header，
// resAdditionHeader表示验证响应的header，resBody表示验证响应的body。
func testRequest(url, reqBody, resBody string, reqHeader, resHeader, resAdditionHeader map[string]string, resCode int) error {
	// 请求的body
	var body bytes.Buffer
	body.WriteString(resBody)
	req, err := http.NewRequest(http.MethodGet, url, &body)
	if err != nil {
		return err
	}
	// 请求的header
	for k, v := range reqHeader {
		req.Header.Add(k, v)
	}
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// 验证响应的status code
	if res.StatusCode != resCode {
		return fmt.Errorf(`"response status code" is "%d" no "%d"`, resCode, res.StatusCode)
	}
	// 验证响应的header
	for k, v := range resHeader {
		if v != res.Header.Get(k) {
			return fmt.Errorf(`"response header"."%s" is "%s" no "%s"`, k, res.Header.Get(k), v)
		}
	}
	for k, v := range resAdditionHeader {
		if v != res.Header.Get(k) {
			return fmt.Errorf(`"response addition header"."%s" is "%s" no "%s"`, k, res.Header.Get(k), v)
		}
	}
	// 验证响应的body
	body.Reset()
	_, err = io.Copy(&body, res.Body)
	if err != nil {
		return err
	}
	if string(body.Bytes()) != resBody {
		return fmt.Errorf(`"response body" is "%s" no "%s"`, body.String(), resBody)
	}
	return nil
}

type testGateway struct {
	*Gateway
	err error
}

func (tg *testGateway) Serve(gatewayAddr, service1Route, service2Route, service1Addr, service2Addr string) {
	testHandlerName := "testHandler"
	// 注册
	handler.RegisterHandler(testHandlerName, func(data map[string]interface{}) (handler.Handler, error) {
		hd := new(testHandler)
		hd.data = data["data"].(string)
		return hd, nil
	})
	// 初始化
	tg.Gateway, tg.err = NewGateway(map[string]interface{}{
		"listen": gatewayAddr,
		"interceptor": []map[string]interface{}{
			{
				"name": testHandlerName,
				"data": "testHandlerInterceptor",
			},
		},
		"notfound": []map[string]interface{}{
			{
				"name": testHandlerName,
				"data": "testHandlerNotfound",
			},
		},
		"handler": map[string]interface{}{
			service1Route: []map[string]interface{}{
				{
					"name": testHandlerName,
					"data": "testHandler",
				},
				{
					"name":                   handler.DefaultHandlerName(),
					"requestUrl":             "http://" + service1Addr,
					"requestHeader":          []string{"req11", "req12"},
					"requestAdditionHeader":  []string{"reqadd1", "reqadd2"},
					"responseAdditionHeader": []string{"resadd1", "resadd2"},
				},
			},
			service2Route: []map[string]interface{}{
				{
					"name":                   handler.DefaultHandlerName(),
					"requestUrl":             "http://" + service2Addr,
					"requestHeader":          []string{"req21", "req22"},
					"requestAdditionHeader":  []string{"reqadd1", "reqadd2"},
					"responseAdditionHeader": []string{"resadd1", "resadd2"},
				},
			},
		},
	})
	if tg.err != nil {
		return
	}
	tg.Gateway.Serve()
}

func Test_Gateway(t *testing.T) {
	// 地址
	gatewayAddr := "127.0.0.1:3390"
	service1Addr := "127.0.0.1:3391"
	service2Addr := "127.0.0.1:3392"
	service1Route := "/service1"
	service2Route := "/service2"
	service1Path := "/path1"
	service2Path := "/"
	// header
	reqHeader1 := make(map[string]string)
	reqHeader1["req11"] = "11"
	reqHeader1["req12"] = "12"
	resHeader1 := make(map[string]string)
	resHeader1["res11"] = "res11"
	resHeader1["res12"] = "res12"
	reqHeader2 := make(map[string]string)
	reqHeader2["req21"] = "req21"
	reqHeader2["req22"] = "req22"
	resHeader2 := make(map[string]string)
	resHeader2["res21"] = "res21"
	resHeader2["res22"] = "res22"
	// addition header
	reqAdditionHeader := make(map[string]string)
	reqAdditionHeader["reqadd1"] = "reqadd1"
	reqAdditionHeader["reqadd2"] = "reqadd2"
	resAdditionHeader := make(map[string]string)
	resAdditionHeader["resadd1"] = "resadd1"
	resAdditionHeader["resadd2"] = "resadd2"
	// body
	reqBody1 := "req1 body"
	resBody1 := "res1 body"
	reqBody2 := "req2 body"
	resBody2 := "res2 body"
	// status code
	resCode1 := 200
	resCode2 := 202
	// 服务
	var service1, service2 testService
	var serviceWaitGroup sync.WaitGroup
	serviceWaitGroup.Add(2)
	go func() {
		defer serviceWaitGroup.Done()
		service1.Serve(service1Addr, service1Path, reqBody1, resBody1, reqHeader1, reqAdditionHeader, resHeader1, resCode1)
	}()
	go func() {
		defer serviceWaitGroup.Done()
		service2.Serve(service2Addr, service2Path, reqBody2, resBody2, reqHeader2, reqAdditionHeader, resHeader2, resCode2)
	}()
	// 网关
	var gateway testGateway
	var gatewayWaitGroup sync.WaitGroup
	var gatewayError error
	go func() {
		defer gatewayWaitGroup.Done()
		gateway.Serve(gatewayAddr, service1Route, service2Route, service1Addr, service2Addr)
	}()
	// 请求
	var req1err, req2err error
	go func() {
		// 关闭service
		defer func() {
			service1.Close()
		}()
		// 等待1s serivce ok
		time.Sleep(time.Second)
		// service1
		req1err = testRequest("http://"+gatewayAddr+service1Route+service1Path, reqBody1, resBody1, reqHeader1, resHeader1, resAdditionHeader, resCode1)
	}()
	go func() {
		// 关闭service
		defer func() {
			service2.Close()
		}()
		// 等待1s serivce ok
		time.Sleep(time.Second)
		// service2
		req2err = testRequest("http://"+gatewayAddr+service2Route+service2Path, reqBody2, resBody2, reqHeader2, resHeader2, resAdditionHeader, resCode2)
	}()
	serviceWaitGroup.Done()
	gateway.Close()
	if req1err != nil {
		t.Fatal(req1err)
	}
	if req2err != nil {
		t.Fatal(req2err)
	}
	if gatewayError != nil {
		t.Fatal(gatewayError)
	}
	if service1.err != nil {
		t.Fatal(service1.err)
	}
	if service2.err != nil {
		t.Fatal(service2.err)
	}
}
