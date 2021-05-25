package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/qq51529210/redis"
)

var (
	// AuthenticationInterceptor register name.
	authenticationInterceptorRegisterName = HandlerName(&AuthenticationInterceptor{})
)

func init() {
	// Register AuthenticationInterceptor.
	RegisterHandler(authenticationInterceptorRegisterName, NewAuthenticationInterceptor)
}

// Get AuthenticationInterceptor register name.
func AuthenticationInterceptorRegisterName() string {
	return authenticationInterceptorRegisterName
}

// A Interceptor that handle authentication.
// It'll check cookie and header["Authorization"].
type AuthenticationInterceptor struct {
	InterceptData
	// Cookie name.
	CookieName string
	// Redis client
	redis *redis.Client
}

func (h *AuthenticationInterceptor) Release() {
	if h.redis != nil {
		h.redis.Close()
	}
}

func (h *AuthenticationInterceptor) Handle(c *Context) bool {
	// 1. Cookie
	if h.CookieName != "" {
		cookie, _ := c.Req.Cookie(h.CookieName)
		// Has cookie
		if cookie != nil {
			_, err := h.redis.Cmd("GET", cookie.Value)
			if err == nil {
				return true
			}
		}
	}
	// 2. Header,"Authorization: Bearer xxxx"
	str := c.Req.Header.Get("Authorization")
	if str != "" {
		const bearerTokenPrefix = "Bearer "
		if strings.HasPrefix(str, bearerTokenPrefix) {
			_, err := h.redis.Cmd("GET", str[len(bearerTokenPrefix):])
			if err == nil {
				return true
			}
		}
	}
	// 3. Token not found.
	h.InterceptData.WriteToResponse(c.Res)
	return false
}

func (h *AuthenticationInterceptor) Update(data interface{}) error {
	d, ok := data.(*AuthenticationInterceptorData)
	if !ok {
		return errors.New(`data must be "*AuthenticationInterceptorData" type`)
	}
	h.InterceptData = d.InterceptData
	h.InterceptData.Check(http.StatusUnauthorized)
	if d.Redis != nil {
		if h.redis != nil {
			h.redis.Close()
		}
		h.redis = redis.NewClient(nil, d.Redis)
	}
	if h.CookieName == "" {
		h.CookieName = "token"
	}
	return nil
}

type AuthenticationInterceptorData struct {
	InterceptData
	Redis      *redis.ClientConfig `json:"redis"`
	CookieName string              `json:"cookieName"`
}

// Create a new AuthenticationInterceptor
func NewAuthenticationInterceptor(data interface{}) (Handler, error) {
	var d *AuthenticationInterceptorData
	switch v := data.(type) {
	case *AuthenticationInterceptorData:
		d = v
	case string:
		d = new(AuthenticationInterceptorData)
		err := json.Unmarshal([]byte(v), d)
		if err != nil {
			return nil, err
		}
	case map[string]interface{}:
		err := Map2Struct(v, d)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid data type %s", reflect.TypeOf(data))
	}
	h := new(AuthenticationInterceptor)
	err := h.Update(d)
	if err != nil {
		return nil, err
	}
	return h, nil
}
