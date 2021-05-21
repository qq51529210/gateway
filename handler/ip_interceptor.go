package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
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

// IPInterceptor更新的数据
type IPInterceptorUpdateData struct {
	Add    []string `json:"add"`
	Remove []string `json:"remove"`
	Data   string   `json:"data"`
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

// 实现接口，更新ip列表。
func (h *IPInterceptor) Update(data interface{}) error {
	// 解析
	d, ok := data.(*IPInterceptorUpdateData)
	if !ok {
		return errors.New(`data must be "*IPInterceptorUpdateData" type`)
	}
	// add
	for i, a := range d.Add {
		_, err := net.ResolveIPAddr("ip", a)
		if err != nil {
			return fmt.Errorf(`"add[%d]" %s`, i, err.Error())
		}
		h.IP.Store(a, 1)
	}
	// remove
	for i, a := range d.Remove {
		_, err := net.ResolveIPAddr("ip", a)
		if err != nil {
			return fmt.Errorf(`"remove[%d]" %s`, i, err.Error())
		}
		h.IP.Delete(a)
	}
	if d.Data != "" {
		h.Data = []byte(d.Data)
	}
	return nil
}

// 实现接口
func (h *IPInterceptor) Name() string {
	return ipInterceptorRegisterName
}

// 创建新的ip拦截器，已经注册。data的格式为
// {
// 	"name": "github.com/qq51529210/handler/IPInterceptor",
// 	"data": "[\"ip1\", \"ip2\"]"
// }
func NewIPAddrInterceptor(data *NewHandlerData) (Handler, error) {
	// 解析
	var d []string
	err := json.Unmarshal([]byte(data.Data), &d)
	if err != nil {
		return nil, err
	}
	h := new(IPInterceptor)
	for i, a := range d {
		_, err := net.ResolveIPAddr("ip", a)
		if err != nil {
			return nil, fmt.Errorf(`"data[%d]" %s`, i, err.Error())
		}
		h.IP.Store(a, 1)
	}
	return h, nil
}
