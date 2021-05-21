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
	c.Req.Header.Add(h.data, h.data)
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

func testService(ser *http.Server, address, path, query, reqBody, resBody string, reqHeader, reqAdditionHeader, resHeader []string, resCode int) error {
	var err error
	ser.Addr = address
	ser.Handler = http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		// 验证请求路径
		if req.URL.Path != path {
			err = fmt.Errorf(`%s "request path" is "%s" no "%s"`, address, req.URL.Path, path)
			return
		}
		// 验证请求参数
		if req.URL.RawQuery != query {
			err = fmt.Errorf(`%s "request query" is "%s" no "%s"`, address, req.URL.RawQuery, query)
			return
		}
		// 验证请求的header
		for _, k := range reqHeader {
			if req.Header.Get(k) != k {
				err = fmt.Errorf(`%s "request header"."%s" is "%s" no "%s"`, address, k, req.Header.Get(k), k)
				return
			}
		}
		for _, k := range reqAdditionHeader {
			if req.Header.Get(k) != k {
				err = fmt.Errorf(`%s "request addition header"."%s" is "%s" no "%s"`, address, k, req.Header.Get(k), k)
				return
			}
		}
		// 验证请求的body
		var body strings.Builder
		_, err = io.Copy(&body, req.Body)
		if err != nil {
			return
		}
		if body.String() != reqBody {
			err = fmt.Errorf(`%s "request body" is "%s" no "%s"`, address, body.String(), reqBody)
			return
		}
		// 响应
		for _, k := range resHeader {
			res.Header().Add(k, k)
		}
		res.WriteHeader(resCode)
		_, err = io.WriteString(res, resBody)
	})
	ser.ListenAndServe()
	return err
}

func testNewGateway(gatewayAddr string, serviceAddr, serviceRoute, handlerData, reqHeader1, reqHeader2, reqAdditionHeader, resAdditionHeader []string) (*Gateway, error) {
	testHandlerName := "testHandler"
	// 注册
	handler.RegisterHandler(testHandlerName, func(data *handler.NewHandlerData) (handler.Handler, error) {
		hd := new(testHandler)
		hd.data = data.Data
		return hd, nil
	})
	// 网关
	return NewGateway(&NewGatewayData{
		Listen: gatewayAddr,
		Handler: map[string][]*handler.NewHandlerData{
			serviceRoute[0]: {
				{
					Name: testHandlerName,
					Data: handlerData[0],
				},
				{
					Name: handler.DefaultHandlerName(),
					Data: testNewDefaultHandlerDataString(serviceAddr[0], reqHeader1, reqAdditionHeader, resAdditionHeader),
				},
			},
			serviceRoute[1]: {
				{
					Name: testHandlerName,
					Data: handlerData[1],
				},
				{
					Name: handler.DefaultHandlerName(),
					Data: testNewDefaultHandlerDataString(serviceAddr[1], reqHeader2, reqAdditionHeader, resAdditionHeader),
				},
			},
		},
	})
}

func testNewDefaultHandlerDataString(serviceAddr string, reqHeader, reqAdditionHeader, resAdditionHeader []string) string {
	var data handler.DefaultHandlerData
	data.RequestUrl = "http://" + serviceAddr
	data.RequestHeader = reqHeader
	data.RequestAdditionHeader = make(map[string]string)
	for _, k := range reqAdditionHeader {
		data.RequestAdditionHeader[k] = k
	}
	data.ResponseAdditionHeader = make(map[string]string)
	for _, k := range resAdditionHeader {
		data.ResponseAdditionHeader[k] = k
	}
	var str strings.Builder
	json.NewEncoder(&str).Encode(&data)
	return str.String()
}

func testRequest(url, reqBody, resBody string, reqHeader, resHeader, resAdditionHeader []string, resCode int) error {
	// 请求的body
	var body bytes.Buffer
	body.WriteString(reqBody)
	req, err := http.NewRequest(http.MethodGet, url, &body)
	if err != nil {
		return err
	}
	// 请求的header
	for _, k := range reqHeader {
		req.Header.Add(k, k)
	}
	// 请求
	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	// 验证响应的status code
	if res.StatusCode != resCode {
		return fmt.Errorf(`"response status code" is "%d" no "%d"`, res.StatusCode, resCode)
	}
	// 验证响应的header
	for _, k := range resHeader {
		if res.Header.Get(k) != k {
			return fmt.Errorf(`"response header"."%s" is "%s" no "%s"`, k, res.Header.Get(k), k)
		}
	}
	for _, k := range resAdditionHeader {
		if res.Header.Get(k) != k {
			return fmt.Errorf(`"response addition header"."%s" is "%s" no "%s"`, k, res.Header.Get(k), k)
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

func Test_Gateway(t *testing.T) {
	// 地址
	gatewayAddr := "127.0.0.1:3390"
	serviceAddr := []string{"127.0.0.1:3391", "127.0.0.1:3392"}
	serviceRoute := []string{"/service1", "/service2"}
	reqPath := []string{"/path1", "/"}
	reqQuery := []string{"a=1&b=2", ""}
	reqHeader := [][]string{{"req11", "req12"}, {"req21", "req22"}}
	resHeader := [][]string{{"res11", "res12"}, {"res21", "res22"}}
	reqAddHeader := []string{"reqAdd11", "reqAdd12"}
	resAddHeader := []string{"resAdd11", "resAdd12"}
	reqBody := []string{"req1 body", "req2 body"}
	resBody := []string{"res1 body", "res2 body"}
	resCode := []int{401, 402}
	reqHandler := []string{"handler1", "handler2"}
	var serviceWaitGroup sync.WaitGroup
	serviceWaitGroup.Add(3)
	// 网关
	gw, err := testNewGateway(gatewayAddr, serviceAddr, serviceRoute, reqHandler, reqHeader[0], reqHeader[1], append(reqAddHeader, reqHandler...), resAddHeader)
	if err != nil {
		t.Fatal(err)
	}
	// 服务
	var service [2]http.Server
	var serviceErr [2]error
	for i := 0; i < 2; i++ {
		go func(i int) {
			defer serviceWaitGroup.Done()
			serviceErr[i] = testService(&service[i], serviceAddr[i], reqPath[i], reqQuery[i], reqBody[i], resBody[i], append(reqHeader[i], reqHandler[i]), reqAddHeader, resHeader[i], resCode[i])
		}(i)
	}
	go func() {
		defer serviceWaitGroup.Done()
		gw.Serve()
	}()
	// 请求
	time.Sleep(time.Millisecond * 100)
	for i := 0; i < 2; i++ {
		url := "http://" + gatewayAddr + serviceRoute[i]
		if reqPath[i] != "/" {
			url += reqPath[i]
		}
		if reqQuery[i] != "" {
			url += "?" + reqQuery[i]
		}
		err = testRequest(url, reqBody[i], resBody[i], reqHeader[i], resHeader[i], resAddHeader, resCode[i])
		if err != nil {
			t.Fatal(err)
		}
		service[i].Close()
	}
	gw.Close()
	serviceWaitGroup.Wait()
	for i := 0; i < 2; i++ {
		if serviceErr[i] != nil {
			t.Fatal(serviceErr[i])
		}
	}
}
