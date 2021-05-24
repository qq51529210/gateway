package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/qq51529210/redis"
)

var (
	// IPInterceptor register name.
	ipInterceptorRegisterName = HandlerName(&IPInterceptor{})
)

func init() {
	// Register IPInterceptor.
	RegisterHandler(ipInterceptorRegisterName, NewIPInterceptor)
}

// Get IPInterceptor register name.
func IPInterceptorRegisterName() string {
	return ipInterceptorRegisterName
}

// Use for intercept request by ip address.
// Request's ip in redis will be intercepted.
type IPInterceptor struct {
	InterceptData
	// Redis client
	redis *redis.Client
}

func (h *IPInterceptor) Handle(c *Context) bool {
	// Split ip address and port.
	i := strings.IndexByte(c.Req.RemoteAddr, ':')
	if i < 1 {
		return false
	}
	// Redis
	value, err := h.redis.Cmd("get", c.Req.RemoteAddr[:i])
	if err != nil || value == nil {
		h.InterceptData.WriteToResponse(c.Res)
		return false
	}
	return true
}

type IPInterceptorData struct {
	Redis redis.ClientConfig `json:"redis"`
	InterceptData
}

// Update IPIntercepto,data is *IPInterceptorData
func (h *IPInterceptor) Update(data interface{}) error {
	d, ok := data.(*IPInterceptorData)
	if !ok {
		return errors.New(`data must be "*IPInterceptorUpdateData" type`)
	}
	h.InterceptData = d.InterceptData
	h.InterceptData.Check(http.StatusForbidden)
	if h.redis != nil {
		h.redis.Close()
	}
	h.redis = redis.NewClient(nil, &d.Redis)
	return nil
}

func (h *IPInterceptor) Name() string {
	return ipInterceptorRegisterName
}

// Create a new IPInterceptor
func NewIPInterceptor(data *NewHandlerData) (Handler, error) {
	var d IPInterceptorData
	err := json.Unmarshal([]byte(data.Data), &d)
	if err != nil {
		return nil, err
	}
	h := new(IPInterceptor)
	err = h.Update(&d)
	if err != nil {
		return nil, err
	}
	return h, nil
}
