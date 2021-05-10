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

// ip地址拦截
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

// 添加拦截的ip地址
func (ipi *IPAddrInterceptor) Add(address string) {
	ipi.IP.LoadOrStore(address, 1)
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
		return nil, errors.New(`"ipAddrInterceptor" value must be "[]string"`)
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
