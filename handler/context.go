package handler

import (
	"net/http"
)

// The data passed in the Handler call chain.
type Context struct {
	Res         http.ResponseWriter
	Req         *http.Request
	Path        string      // Route path.
	Data        interface{} // Used for save and pass temp data in call chain.
	Interceptor []Handler   // Gateway interceptor handler list.
	NotFound    []Handler   // Gateway notfound handler list.
	Forward     []Handler   // Gateway forward handler list.
}
