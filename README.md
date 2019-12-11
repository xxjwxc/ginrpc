## Automatic parameter binding base on [go-gin](https://github.com/gin-gonic/gin)

## [中文文档](README_cn.md)

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 

- Support for RPC automatic mapping
- Support object registration
- Support annotation routing
- base on [go-gin](https://github.com/gin-gonic/gin) on json restful style 
- implementation of parameter filtering and binding with request
- code registration simple and supports multiple ways of registration

### directory structure description

- ginrpc/base/common.go Base Library
- ginrpc/base/api/context.go customize context content
- Supporting Automatic Detection of Parameters binding:"required"  [validator](go-playground/validator.v8)
- Support RPC automatic mapping

### Support three of interface modes

- func(*gin.Context) //gin Primitive interface
  func(*api.Context) //Custom context type
- func(*api.Context,req) //Custom context type,Request parameters with req
  func(*api.Context,req)
- func(*gin.Context,*req)//go-gin context ,Request parameters with req
  func(*gin.Context,req)


### Sample code

## init(sample mod is ginweb )
	``` go mod init ginweb ```

### coding (detailed address：https://github.com/xxjwxc/ginrpc/tree/master/sample/ginweb)
```go
package main

import (
	"fmt"
	"net/http"

	_ "ginweb/routers" // debug模式需要添加[mod]/routers 注册注解路由

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

type ReqTest struct {
	Access_token string `json:"access_token"`
	UserName     string `json:"user_name" binding:"required"` // 带校验方式
	Password     string `json:"password"`
}

// Hello ...
type Hello struct {
	Index int
}

// Hello 带注解路由(参考beego形式)
// @router /block [post,get]
func (s *Hello) Hello(c *api.Context, req *ReqTest) {
	fmt.Println(req)
	fmt.Println(s.Index)
	c.JSON(http.StatusOK, "ok")
}

// Hello2 不带注解路由(参数为2默认post)
func (s *Hello) Hello2(c *gin.Context, req ReqTest) {
	fmt.Println(req)
	fmt.Println(s.Index)
	c.JSON(http.StatusOK, "ok")
}

//TestFun1 gin 默认的函数回调地址
func TestFun1(c *gin.Context) {
	fmt.Println(c.Params)
	c.String(200, "ok")
}

//TestFun2 自定义context的函数回调地址
func TestFun2(c *api.Context) {
	fmt.Println(c.Params)
	c.JSON(http.StatusOK, "ok")
}

//TestFun3 带自定义context跟已解析的req参数回调方式
func TestFun3(c *api.Context, req *ReqTest) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

//TestFun4 带自定义context跟已解析的req参数回调方式
func TestFun4(c *gin.Context, req ReqTest) {
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}

func main() {
	base := ginrpc.New(ginrpc.WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), ginrpc.WithDebug(true), ginrpc.WithGroup("xxjwxc"))

	router := gin.Default()
	h := new(Hello)
	h.Index = 123
	base.Register(router, h)                          // 对象注册
	router.POST("/test1", base.HandlerFunc(TestFun1)) // 函数注册
	router.POST("/test2", base.HandlerFunc(TestFun2))
	router.POST("/test3", base.HandlerFunc(TestFun3))
	router.POST("/test4", base.HandlerFunc(TestFun4))
	base.RegisterHandlerFunc(router, []string{"post", "get"}, "/test", TestFun1) // 多种请求方式注册

	router.Run(":8080")
}
   ```

- curl
  ```
  curl 'http://127.0.0.1:8080/test4' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

### Annotation routing

- 1.Annotation route will automatically create[mod]/routers/gen_router.go file and   which needs to be added when calling：
	```
	_ "[mod]/routers" // Debug mode requires adding [mod]/routes to register annotation routes
	```
	By default, the [gen_router. Data] file will also be generated in the root directory of the project (keep the secondary file, and you can embed it without adding the above code)

- 2.Annotation route call mode:
	```
	base := ginrpc.New(ginrpc.WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), ginrpc.WithDebug(true), ginrpc.WithGroup("xxjwxc"))
	base.Register(router, new(Hello))                          // 对象注册
	router.Run(":8080")
	```
	more demo  [ginweb](/sample/ginweb)
- 3.Execute curl to automatically bind parameters. See the results directly
  ```
  curl 'http://127.0.0.1:8080/xxjwxc/block' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```
  ```
  curl 'http://127.0.0.1:8080/xxjwxc/hello.hello2' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```
- 4 Parameter description
	ginrpc.WithCtx ： Set custom context
	ginrpc.WithDebug(true) : set debug style
	ginrpc.WithGroup("xxjwxc") : Add routing prefix (you can also use gin. Group grouping)
	ginrpc.WithBigCamel(true) : Set big hump standard (false is web mode, _, lowercase)

	[more](https://godoc.org/github.com/xxjwxc/ginrpc)

### coding address： [ginprc](https://github.com/xxjwxc/ginrpc) Please give star support