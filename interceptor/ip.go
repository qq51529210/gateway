package interceptor

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	gateway "github.com/qq51529210/gateway"
)

var (
	ipInterceptorRegisterName = gateway.HandlerName(&IPInterceptor{}) // 注册名称
)

func init() {
	// 注册
	gateway.RegisterHandler(ipInterceptorRegisterName, NewIPAddrInterceptor)
}

// 获取IPInterceptor的注册名称
func IPInterceptorRegisterName() string {
	return ipInterceptorRegisterName
}

// 基于本地内存的ip地址拦截
type IPInterceptor struct {
	IP sync.Map
}

// 实现接口
func (ipi *IPInterceptor) Handle(c *gateway.Context) bool {
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
func (h *IPInterceptor) Update(data interface{}) error {
	m, ok := data.(map[string]interface{})
	if !ok {
		return errors.New(`data must be "map[string][]string" type`)
	}
	value, ok := m["add"]
	if ok {
		array, ok := value.([]interface{})
		if !ok {
			return errors.New(`"add" data must be "[]string" type`)
		}
		for i, a := range array {
			str, ok := a.(string)
			if !ok {
				return fmt.Errorf(`"add" [%d] must be "string" type`, i)
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
			return errors.New(`"remove" data must be "[]string" type`)
		}
		for i, a := range array {
			str, ok := a.(string)
			if !ok {
				return fmt.Errorf(`"remove" [%d] must be "string" type`, i)
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

// 实现接口
func (h *IPInterceptor) Name() string {
	return ipInterceptorRegisterName
}

// 创建新的ip拦截器，data的json格式
// [
// 		"ip1",
//		"ip2",
//		...
// ]
func NewIPAddrInterceptor(data interface{}) (gateway.Handler, error) {
	value, ok := data.([]interface{})
	if !ok {
		return nil, errors.New(`"IPInterceptor" data must be "[]string" type`)
	}
	ip := new(IPInterceptor)
	for i, v := range value {
		// 检查类型
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf(`"IPInterceptor" [%d] must be "string" type`, i)
		}
		// 检查格式
		addr, err := net.ResolveIPAddr("ip", s)
		if err != nil {
			return nil, fmt.Errorf(`"IPInterceptor" [%d] %s`, i, err.Error())
		}
		// 保存
		ip.IP.Store(addr.IP.String(), 1)
	}
	return ip, nil
}
