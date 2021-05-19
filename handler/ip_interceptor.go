package handler

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/qq51529210/gateway/util"
)

var (
	ipInterceptorRegisterName = HandlerName(&IPInterceptor{}) // 注册名称
)

func init() {
	RegisterHandler(ipInterceptorRegisterName, NewIPAddrInterceptor) // 注册
}

// 获取IPInterceptor的注册名称
func IPInterceptorRegisterName() string {
	return ipInterceptorRegisterName
}

// 基于内存的ip地址拦截器，类似防止爬虫的需要智能增删，这个不适合。
type IPInterceptor struct {
	IP   sync.Map // ip地址表，key:string
	Data []byte   // 返回的body数据
}

// 实现接口
func (h *IPInterceptor) Handle(c *Context) bool {
	// 分开地址和端口
	i := strings.IndexByte(c.Req.RemoteAddr, ':')
	if i < 1 {
		return false
	}
	// 检查
	_, ok := h.IP.Load(c.Req.RemoteAddr[:i])
	if ok {
		// 存在，返回403和数据
		c.Res.WriteHeader(http.StatusForbidden)
		c.Res.Write(h.Data)
	}
	return !ok
}

// 实现接口，更新ip列表，data的json格式：
// {
// 	"add": [
// 		"",
// 		"",
// 		...
// 	],
// 	"remove": [
// 		"ip1",
// 		"ip2",
// 		...
// 	],
// 	"data": ""
// }
// add表示添加的ip地址， remove表示移除ip地址，data表示相应的数据。
func (h *IPInterceptor) Update(data interface{}) error {
	initData, ok := data.(map[string]interface{})
	if !ok {
		return errors.New(`data must be "map[string][]string" type`)
	}
	// add
	array, err := util.Data(initData).StringSlice("add")
	if err != nil {
		return err
	}
	for i, a := range array {
		_, err := net.ResolveIPAddr("ip", a)
		if err != nil {
			return fmt.Errorf(`"add[%d]" %s`, i, err.Error())
		}
		h.IP.Store(a, 1)
	}
	// remove
	array, err = util.Data(initData).StringSlice("remove")
	if err != nil {
		return err
	}
	for i, a := range array {
		_, err := net.ResolveIPAddr("ip", a)
		if err != nil {
			return fmt.Errorf(`"remove[%d]" %s`, i, err.Error())
		}
		h.IP.Delete(a)
	}
	return nil
}

// 实现接口
func (h *IPInterceptor) Name() string {
	return ipInterceptorRegisterName
}

// 创建新的ip拦截器，data的json格式：
// {
// 	"add":[
// 		"",
// 		"",
// 		...,
// 	],
// 	"data":""
// }
func NewIPAddrInterceptor(data map[string]interface{}) (Handler, error) {
	ip := new(IPInterceptor)
	err := ip.Update(data)
	if err != nil {
		return nil, err
	}
	return ip, nil
}
