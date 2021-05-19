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

	"github.com/goccy/go-json"
	"github.com/qq51529210/gateway/handler"
)

type testHandler struct {
	data string
}

func (h *testHandler) Handle(c *handler.Context) bool {
	c.Req.Header.Add("test", h.data)
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
	if body.String() != resBody {
		return fmt.Errorf(`"response body" is "%s" no "%s"`, body.String(), resBody)
	}
	return nil
}

type testGateway struct {
	*Gateway
	err error
}

func (tg *testGateway) Serve(gatewayAddr string, serviceRoute, serviceAddr, handlerData []string, reqHeader1, reqHeader2, reqAdditionHeader, resAdditionHeader map[string]string) {
	testHandlerName := "testHandler"
	// 注册
	handler.RegisterHandler(testHandlerName, func(data *handler.NewHandlerData) (handler.Handler, error) {
		hd := new(testHandler)
		hd.data = data.Data
		return hd, nil
	})
	var d1, d2 handler.DefaultHandlerData
	d1.RequestUrl = "http://" + serviceAddr[0]
	d1.RequestAdditionHeader = reqAdditionHeader
	d1.ResponseAdditionHeader = resAdditionHeader
	for k := range reqHeader1 {
		d1.RequestHeader = append(d1.RequestHeader, k)
	}
	d2.RequestUrl = "http://" + serviceAddr[1]
	d2.RequestAdditionHeader = reqAdditionHeader
	d2.ResponseAdditionHeader = resAdditionHeader
	for k := range reqHeader2 {
		d1.RequestHeader = append(d2.RequestHeader, k)
	}
	var data []byte
	data, tg.err = json.Marshal(&d1)
	if tg.err != nil {
		return
	}
	data1 := string(data)
	data, tg.err = json.Marshal(&d2)
	if tg.err != nil {
		return
	}
	data2 := string(data)
	// 初始化
	tg.Gateway, tg.err = NewGateway(&NewGatewayData{
		Listen: gatewayAddr,
		Handler: map[string][]*handler.NewHandlerData{
			serviceRoute[0]: {
				{
					Name: testHandlerName,
					Data: handlerData[0],
				},
				{
					Name: handler.DefaultHandlerName(),
					Data: data1,
				},
			},
			serviceRoute[1]: {
				{
					Name: testHandlerName,
					Data: handlerData[1],
				},
				{
					Name: handler.DefaultHandlerName(),
					Data: data2,
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
	serviceAddr := []string{"127.0.0.1:3391", "127.0.0.1:3392"}
	reqPath := []string{"/path1", "/"}
	reqHeader := []map[string]string{{"req11": "req11", "req12": "req12"}, {"req21": "req21", "req22": "req22"}}
	resHeader := []map[string]string{{"res11": "res11", "res12": "res12"}, {"res21": "res21", "res22": "res22"}}
	reqAddHeader := map[string]string{"reqAdd11": "reqAdd11", "reqAdd12": "reqAdd12"}
	resAddHeader := map[string]string{"resAdd11": "resAdd11", "resAdd12": "resAdd12"}
	reqBody := []string{"req1 body", "req2 body"}
	resBody := []string{"res1 body", "res2 body"}
	resCode := []int{200, 300}
	reqHandler := []string{"handler1", "handler2"}
	// 服务
	var service1, service2 testService
	var serviceWaitGroup sync.WaitGroup
	serviceWaitGroup.Add(2)
	go func() {
		defer serviceWaitGroup.Done()
		h := make(map[string]string)
		for k, v := range reqHeader[0] {
			h[k] = v
		}
		h[reqHandler[0]] = reqHandler[0]
		service1.Serve(serviceAddr[0], reqPath[0], reqBody[0], resBody[0], h, reqAddHeader, resHeader[0], resCode[0])
	}()
	go func() {
		defer serviceWaitGroup.Done()
		h := make(map[string]string)
		for k, v := range reqHeader[1] {
			h[k] = v
		}
		h[reqHandler[1]] = reqHandler[1]
		service1.Serve(serviceAddr[1], reqPath[1], reqBody[1], resBody[1], h, reqAddHeader, resHeader[1], resCode[1])
	}()
	// 网关
	serviceRoute := []string{"/service1", "/service2"}
	var gateway testGateway
	go gateway.Serve(gatewayAddr, serviceRoute, serviceAddr, reqHandler, reqHeader[0], reqHeader[1], reqAddHeader, resAddHeader)
	// 请求
	var req1err, req2err error
	go func() {
		// 关闭service
		defer func() {
			service1.Close()
		}()
		// 等待1s serivce ok
		time.Sleep(time.Millisecond * 100)
		// service1
		req1err = testRequest("http://"+gatewayAddr+serviceRoute[0]+reqPath[0], reqBody[0], resBody[0], reqHeader[0], resHeader[0], resAddHeader, resCode[0])
	}()
	go func() {
		// 关闭service
		defer func() {
			service2.Close()
		}()
		// 等待1s serivce ok
		time.Sleep(time.Millisecond * 100)
		// service2
		req1err = testRequest("http://"+gatewayAddr+serviceRoute[1]+reqPath[1], reqBody[1], resBody[1], reqHeader[1], resHeader[1], resAddHeader, resCode[1])
	}()
	serviceWaitGroup.Wait()
	if req1err != nil {
		t.Fatal(req1err)
	}
	if req2err != nil {
		t.Fatal(req2err)
	}
	if gateway.err != nil {
		t.Fatal(gateway.err)
	}
	if service1.err != nil {
		t.Fatal(service1.err)
	}
	if service2.err != nil {
		t.Fatal(service2.err)
	}
}
