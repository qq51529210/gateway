package interceptor

import (
	"testing"
)

func Test_IPInterceptor(t *testing.T) {

}

func Test_IPInterceptor_Name(t *testing.T) {
	var h IPInterceptor
	name := h.Name()
	if name != "github.com/qq51529210/gateway/interceptor.IPInterceptor" {
		t.FailNow()
	}
}
