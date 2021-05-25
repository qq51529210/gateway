package handler

import (
	"net/http"
)

// The data passed in Handler call chain.
type Context struct {
	Res http.ResponseWriter
	Req *http.Request
	// Service route path.
	// Top dir of request url path.
	Path string
	// Used for save and pass temp data in Handler call chain.
	Data interface{}
	// Final request
	Forward http.Request
}
