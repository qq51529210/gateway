package handler

import (
	"encoding/json"
	"errors"
	"net/http"
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
	return ipInterceptorRegisterName
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
	d, ok := data.(*IPInterceptorData)
	if !ok {
		return errors.New(`data must be "*AuthenticationInterceptorData" type`)
	}
	h.InterceptData = d.InterceptData
	h.InterceptData.Check(http.StatusUnauthorized)
	if h.redis != nil {
		h.redis.Close()
	}
	h.redis = redis.NewClient(nil, &d.Redis)
	if h.CookieName == "" {
		h.CookieName = "token"
	}
	return nil
}

func (h *AuthenticationInterceptor) Name() string {
	return ipInterceptorRegisterName
}

type AuthenticationInterceptorData struct {
	InterceptData
	Redis      redis.ClientConfig `json:"redis"`
	CookieName string             `json:"cookieName"`
}

func NewAuthenticationInterceptor(data *NewHandlerData) (Handler, error) {
	var d AuthenticationInterceptorData
	err := json.Unmarshal([]byte(data.Data), &d)
	if err != nil {
		return nil, err
	}
	h := new(AuthenticationInterceptor)
	err = h.Update(&d)
	if err != nil {
		return nil, err
	}
	return h, nil
}
