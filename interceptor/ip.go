package interceptor

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	gateway "github.com/qq51529210/gateway"
)

func init() {
	gateway.RegisterHandler("ipAddrInterceptor", NewIPAddrInterceptor)
}

// 基于本地内存的ip地址拦截
type IPAddrInterceptor struct {
	IP sync.Map
}

// 实现接口
func (ipi *IPAddrInterceptor) Handle(c *gateway.Context) bool {
	i := strings.IndexByte(c.Req.RemoteAddr, ':')
	if i < 1 {
		return false
	}
	_, ok := ipi.IP.Load(c.Req.RemoteAddr[:i])
	return !ok
}

// 实现接口
// 更新ip列表，data的json格式
// {
// 		"add": [
// 			"ip1",
// 			"ip2",
// 			...
// 		],
// 		"remove": [
// 			"ip1",
// 			"ip2",
// 			...
// 		]
// }
func (h *IPAddrInterceptor) Update(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return errors.New(`data must be "map[string][]string"`)
	}
	value, ok := m["add"]
	if ok {
		array, ok := value.([]interface{})
		if !ok {
			return errors.New(`"add" value must be "[]string"`)
		}
		for i, a := range array {
			str, ok := a.(string)
			if !ok {
				return fmt.Errorf(`"add" [%d] must be "string"`, i)
			}
			_, err := net.ResolveIPAddr("ip", str)
			if err != nil {
				return fmt.Errorf(`"add" [%d] %s`, i, err)
			}
			h.IP.Store(str, 1)
		}
	}
	value, ok = m["remove"]
	if ok {
		array, ok := value.([]interface{})
		if !ok {
			return errors.New(`"remove" value must be "[]string"`)
		}
		for i, a := range array {
			str, ok := a.(string)
			if !ok {
				return fmt.Errorf(`"remove" [%d] must be "string"`, i)
			}
			_, err := net.ResolveIPAddr("ip", str)
			if err != nil {
				return fmt.Errorf(`"remove" [%d] %s`, i, err)
			}
			h.IP.Delete(str)
		}
	}
	return nil
}

// 创建新的ip拦截器，data的json格式
// 	"ipAddr": [
//		"ip1",
//		"ip2",
//		...
// ]
func NewIPAddrInterceptor(data interface{}) (gateway.Handler, error) {
	value, ok := data.([]interface{})
	if !ok {
		return nil, errors.New(`"ipAddrInterceptor" data must be "[]string"`)
	}
	ip := new(IPAddrInterceptor)
	for i, v := range value {
		// 检查类型
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf(`"ipAddrInterceptor" [%d] must be string`, i)
		}
		// 检查格式
		addr, err := net.ResolveIPAddr("ip", s)
		if err != nil {
			return nil, fmt.Errorf(`"ipAddrInterceptor" [%d] %s`, i, err.Error())
		}
		// 保存
		ip.IP.Store(addr.IP.String(), 1)
	}
	return ip, nil
}
