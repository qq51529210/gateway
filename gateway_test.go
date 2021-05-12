package gateway

import (
	"testing"
	"time"
)

func Test_Gateway(t *testing.T) {
	gw, err := NewGateway(map[string]interface{}{
		"listen": ":33966",
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		time.Sleep(time.Second)
	}()
	gw.Serve()
}
