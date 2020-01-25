[![Build Status](https://travis-ci.org/xxjwxc/ginrpc.svg?branch=master)](https://travis-ci.org/xxjwxc/ginrpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/xxjwxc/ginrpc)](https://goreportcard.com/report/github.com/xxjwxc/ginrpc)
[![codecov](https://codecov.io/gh/xxjwxc/ginrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/xxjwxc/ginrpc)
[![GoDoc](https://godoc.org/github.com/xxjwxc/ginrpc?status.svg)](https://godoc.org/github.com/xxjwxc/ginrpc)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 

# [中文文档](README_cn.md)

## Automatic parameter binding base on [go-gin](https://github.com/gin-gonic/gin)



## Golang gin automatic parameter binding

- Support for RPC automatic mapping
- Support object registration
- Support annotation routing
- base on [go-gin](https://github.com/gin-gonic/gin) on json restful style
- implementation of parameter filtering and binding with request
- code registration simple and supports multiple ways of registration
- [grpc-go](https://github.com/grpc/grpc-go) bind support


## API details

### Three interface modes are supported

- func(*gin.Context) // go-gin Raw interface

  func(*api.Context) // Custom context type

- func(*api.Context,req) // Custom context type,with request

  func(*api.Context,*req)

- func(*gin.Context,*req) // go-gin context,with request

  func(*gin.Context,req)

- func(*gin.Context,*req)(*resp,error) // go-gin context,with request,return parameter and error ==> [grpc-go](https://github.com/grpc/grpc-go)

  func(*gin.Context,req)(resp,error)

## 一,Parameter auto binding

```go

package main

import (
	"fmt"
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

type ReqTest struct {
	Access_token string `json:"access_token"`
	UserName     string `json:"user_name" binding:"required"` // With verification mode.带校验方式
	Password     string `json:"password"`
}

//TestFun6 Callback method with custom context and resolved req parameters
func TestFun6(c *gin.Context, req ReqTest)  (*ReqTest, error) {
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}

func main() {
	base := ginrpc.New() 
	router := gin.Default()
	router.POST("/test6", base.HandlerFunc(TestFun6))
	base.RegisterHandlerFunc(router, []string{"post", "get"}, "/test", TestFun4) // Multiple request mode registration
	router.Run(":8080")
}

   ```

- curl

  ```
  curl 'http://127.0.0.1:8080/test4' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'

  ```

## 二,Object registration (annotation routing)

### Initialization project (this project is named after `ginweb`)
	``` go mod init ginweb ```

### coding [more>>](https://github.com/xxjwxc/ginrpc/tree/master/sample/ginweb)
```go

package main

import (
	"fmt"
	"net/http"

	_ "ginweb/routers" // Debug mode requires adding [mod] / routes to register annotation routes.debug模式需要添加[mod]/routers 注册注解路由

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

type ReqTest struct {
	Access_token string `json:"access_token"`
	UserName     string `json:"user_name" binding:"required"` // With verification mode
	Password     string `json:"password"`
}

// Hello ...
type Hello struct {
}

// Hello Annotated route (bese on beego way)
// @router /block [post,get]
func (s *Hello) Hello(c *api.Context, req *ReqTest) {
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// Hello2 Route without annotation (the parameter is 2 default post)
func (s *Hello) Hello2(c *gin.Context, req ReqTest) {
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// [grpc-go](https://github.com/grpc/grpc-go)
// with request,return parameter and error
// Hello3 Route without annotation (the parameter is 2 default post)
func (s *Hello) Hello3(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	return &req,nil
}

func main() {
	base := ginrpc.New(ginrpc.WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), ginrpc.WithDebug(true), ginrpc.WithGroup("xxjwxc"))

	router := gin.Default()
	base.Register(router, new(Hello))                          // object register like(go-micro)
	// or base.Register(router, new(Hello)) 
	router.Run(":8080")
}
   ```

### - Annotation routing related instructions

```
 // @router /block [post,get]

@router tag  /block router [post,get] method 

 ```

 #### Note: if there is no annotation route in the object function, the system will add annotation route by default. Post mode: with req (2 parameters (CTX, req)), get mode is a parameter (CTX)



### 1. Annotation route will automatically create `[mod]/routes/gen_router.go` file, which needs to be added when calling:

	```
	_ "[mod]/routers" // Debug mode requires adding [mod] / routes to register annotation routes

	```

	By default, the [gen_router. Data] file will also be generated in the root directory of the project (keep this file, and you can embed it without adding the above code)

### 2. way of annotation route :

	more to saying  [ginweb](/sample/ginweb)

### 3. Parameter description

	ginrpc.WithCtx ： Set custom context

	ginrpc.WithDebug(true) : Set debug mode

	ginrpc.WithGroup("xxjwxc") : Add routing prefix (you can also use gin. Group grouping)

	ginrpc.WithBigCamel(true) : Set big camel standard (false is web mode, _, lowercase)

	[more>>](https://godoc.org/github.com/xxjwxc/ginrpc)

### 4. Execute curl to automatically bind parameters. See the results directly

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/block' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/hello.hello2' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

## Stargazers over time

[![Stargazers over time](https://starchart.cc/xxjwxc/ginrpc.svg)](https://starchart.cc/xxjwxc/ginrpc)
      

## Next

	1. Export API documents

	2. Export postman test configuration

### coding address:[ginprc](https://github.com/xxjwxc/ginrpc) Please give star support