# gateway

A http gateway application written in Golang.
It is designed in plug-in mode,so it is easy to add,update,remove functions.
To add a new function you need to implement interface Handler and call RegisterHandler to register your Handler.
Than you can modify or remove Handler in application runtime.

## Main logic

```go
func ServeHTTP(){
  // First call intercept handler chain.
  // If top directory of request url path is route,call forward handler chain.
  // Else,call notfound handler chain.
}
```

## How to add a new Handler code

```go
var (
	myHandlerRegisterName = HandlerName(&MyHandler{})
)

func init() {
	RegisterHandler(myHandlerRegisterName, NewMyHandler)
}

func NewMyHandler(data interface{}) (Handler, error){
  h:=new(MyHandler)
  h.Update(data)
  return h, nil
}

type MyHandler struct {
  // Fields
}

func (h*MyHandler)Handle(c *Context)bool{
  // Handle request
  return true
}

func (h*MyHandler)Update(data interface{})error{
  // Update MyHandler field
  return nil
}
```

## Create customer handler chain by configure

```go
var cfg NewGatewayData{}
LocaConfig(&cfg)
app := NewGateway(&cfg)
```

## Update handler chain in application runtime

Provide HTTP-API to manage handler chain.

```go
func UpdateIntercept(){
  // Initial intercept handlers.
  // Convert to json.
  // HTTP-API request.
}
```

## Handler list

- [DefaultInterceptor](./handler/handler.go)

  This handler do nothing.

- [DefaultForward](./handler/handler.go)

  Forward request and response.You can specify which request headers to forward,additional headers for request and response.

- [DefaultNotfound](./handler/handler.go)

  Response 404 and message.Both of two can be specified.

- [IPInterceptor](./handler/ip_interceptor.go)

  Itercept request be ip address,response 403 and message.

  Use redis to store ip.

- [AuthenticationInterceptor](./handler/authentication_interceptor.go)

  Check cookie or "Authorization" header for token,if not found,response 401 and message.

  Use redis to store token.

## Other Handler to be implemented.

- Current limiting

- Fusing

## HTTP-API

Provide http-api to manage application runtime handler chain.You should run api server first.

```go
// To run api server
var cfg NewGatewayData{
  "api-listen":""
  "api-token":""
}
app := NewGateway(&cfg)
```

- Intercept

  | path        | method | content-type     | token     | body             |
  | ----------- | ------ | ---------------- | --------- | ---------------- |
  | /intercepts | put    | application/json | api-token | []NewHandlerData |

- NotFound

  | path       | method | content-type     | token     | body             |
  | ---------- | ------ | ---------------- | --------- | ---------------- |
  | /notfounds | put    | application/json | api-token | []NewHandlerData |

- Forward

  | path      | method | content-type     | token     | body             |
  | --------- | ------ | ---------------- | --------- | ---------------- |
  | /forwards | put    | application/json | api-token | []NewHandlerData |
