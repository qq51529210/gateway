package gateway

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
)

// ip地址拦截
type IPAddrInterceptor struct {
	IP sync.Map
}

// 实现接口
func (ipi *IPAddrInterceptor) Intercept(res http.ResponseWriter, req *http.Request) bool {
	i := strings.IndexByte(req.RemoteAddr, ':')
	if i < 1 {
		return false
	}
	_, ok := ipi.IP.Load(req.RemoteAddr[:i])
	return !ok
}

// 添加拦截的ip地址
func (ipi *IPAddrInterceptor) Add(address string) {
	ipi.IP.LoadOrStore(address, 1)
}

// 创建新的ip拦截器，cfg必须是，"ip":["x.x.x.x","x.x.x.x"]
func NewIPAddrInterceptor(cfg map[string]interface{}) (Interceptor, error) {
	ip := new(IPAddrInterceptor)
	val, ok := cfg["ip"]
	if ok {
		// 必须是一个["x.x.x.x","x.x.x.x"]
		val, ok := val.([]interface{})
		if !ok {
			return nil, fmt.Errorf("ip config must be \"ip\":[\"x.x.x.x\",\"x.x.x.x\"]")
		}
		for i, a := range val {
			// 检查类型
			s, ok := a.(string)
			if !ok {
				return nil, fmt.Errorf("ip[%d] must be \"x.x.x.x\"", i)
			}
			// 检查格式
			addr, err := net.ResolveIPAddr("ip", s)
			if err != nil {
				return nil, fmt.Errorf("ip[%d] '%s'", i, err.Error())
			}
			// 保存
			ip.IP.Store(addr.IP.String(), 1)
		}
	}
	return ip, nil
}
