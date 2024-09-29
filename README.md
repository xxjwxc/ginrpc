[![Build Status](https://travis-ci.org/xxjwxc/ginrpc.svg?branch=master)](https://travis-ci.org/xxjwxc/ginrpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/xxjwxc/ginrpc)](https://goreportcard.com/report/github.com/xxjwxc/ginrpc)
[![codecov](https://codecov.io/gh/xxjwxc/ginrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/xxjwxc/ginrpc)
[![GoDoc](https://godoc.org/github.com/xxjwxc/ginrpc?status.svg)](https://godoc.org/github.com/xxjwxc/ginrpc)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 
<a href="https://hellogithub.com/repository/fbc519fa5a2e4e248eabd860a6f404ff" target="_blank"><img src="https://abroad.hellogithub.com/v1/widgets/recommend.svg?rid=fbc519fa5a2e4e248eabd860a6f404ff&claim_uid=6IkmGoxiuN4LWFn&theme=small" alt="Featured｜HelloGitHub" /></a>

# [中文文档](README_cn.md)

## Automatic parameter binding base on [go-gin](https://github.com/gin-gonic/gin)

![img](/image/ginrpc.gif)

## doc 

![doc](/image/ginrpc_doc.gif)


## Golang gin automatic parameter binding

- Support for RPC automatic mapping
- Support object registration
- Support annotation routing
- base on [go-gin](https://github.com/gin-gonic/gin) on json restful style
- implementation of parameter filtering and binding with request
- code registration simple and supports multiple ways of registration
- [grpc-go](https://github.com/grpc/grpc-go) bind support
- Support [swagger](http://editor.swagger.io/) [MORE](https://github.com/xxjwxc/gmsec)
- Support [markdown/mindoc](https://www.iminho.me/) [MORE](https://github.com/xxjwxc/gmsec)
- Support call before and after deal(`ginrpc.WithBeforeAfter`)

- [DEMO](https://github.com/xxjwxc/gmsec)

## Installing
- go mod:
```
go get -u github.com/xxjwxc/ginrpc@master
```

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

## 一. Parameter auto binding,Object registration (annotation routing)

### Initialization project (this project is named after `gmsec`)
	``` go mod init gmsec ```

### coding [more>>](hhttps://github.com/gmsec/gmsec)

```go

package main

import (
	"fmt"
	"net/http"

	_ "gmsec/routers" // Debug mode requires adding [mod] / routes to register annotation routes.debug模式需要添加[mod]/routers 注册注解路由
	"github.com/xxjwxc/public/mydoc/myswagger" // swagger 支持

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
// @Router /block [post,get]
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
// TestFun6 Route without annotation (the parameter is 2 default post)
func TestFun6(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	//c.JSON(http.StatusOK, req)
	return &req, nil
}

func main() {

	// swagger
	myswagger.SetHost("https://localhost:8080")
	myswagger.SetBasePath("gmsec")
	myswagger.SetSchemes(true, false)
	// -----end --
	base := ginrpc.New()
	router := gin.Default() // or router :=  gin.Default().Group("/xxjwxc")
	base.Register(router, new(Hello)) // object register like(go-micro)
	router.POST("/test6", base.HandlerFunc(TestFun6))                            // function register
	base.RegisterHandlerFunc(router, []string{"post", "get"}, "/test", TestFun6) 
	router.Run(":8080")
}
   ```

### - Annotation routing related instructions

```

// @Router /block [post,get]
@Router tag  
/block router 
[post,get] method 

 ```

 - `@Router - [-]` indicates ignore

 #### Note: if there is no annotation route in the object function, the system will add annotation route by default. Post mode: with req (2 parameters (CTX, req)), get mode is a parameter (CTX)



### 1. Annotation route will automatically create `[mod]/routes/gen_router.go` file, which needs to be added when calling:

	```
	_ "[mod]/routers" // Debug mode requires adding [mod] / routes to register annotation routes

	```

	By default, the [gen_router. Data] file will also be generated in the root directory of the project (keep this file, and you can embed it without adding the above code)

### 2. way of annotation route :

	more to saying  [gmsec](https://github.com/gmsec/gmsec)

### 3. Parameter description

	ginrpc.WithCtx ： Set custom context

	ginrpc.WithDebug(true) : Set debug mode
	
	ginrpc.WithOutDoc(true) : output markdown/swagger api doc

	ginrpc.WithBigCamel(true) : Set big camel standard (false is web mode, _, lowercase)

	ginrpc.WithBeforeAfter(&ginrpc.DefaultGinBeforeAfter{}) : Before After call

	[more>>](https://godoc.org/github.com/xxjwxc/ginrpc)

### 4. Execute curl to automatically bind parameters. See the results directly

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/block' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/hello.hello2' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

## 二. swagger/markdown/mindoc Document generation description

```
	ginrpc.WithOutDoc(true) : output markdown/swagger
```

### 1.For object registration 'ginrpc. Register' mode, document export is supported
### 2.Export supports annotation routing, Parameter annotation and default value (` tag '. ` default')
### 3.Default export path:(`/docs/swagger/swagger.json`,`/docs/markdown`)
### 4 struct demo
```
type ReqTest struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
}
```
- [more >>>](https://github.com/xxjwxc/gmsec)

## 三. Support to call Middleware
- using `ginrpc.WithBeforeAfter(&ginrpc.DefaultGinBeforeAfter{})` 
- You can also implement functions (single types) on objects
```go
	// GinBeforeAfter Execute middleware before and after the object call (support adding the object separately from the object in total)
	type GinBeforeAfter interface {
		GinBefore(req *GinBeforeAfterInfo) bool
		GinAfter(req *GinBeforeAfterInfo) bool
	}
```

## Stargazers over time

[![Stargazers over time](https://starchart.cc/xxjwxc/ginrpc.svg)](https://starchart.cc/xxjwxc/ginrpc)


### coding address:[ginprc](https://github.com/xxjwxc/ginrpc) Please give star support