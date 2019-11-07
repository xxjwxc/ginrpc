## Automatic parameter binding tool base on [go-gin](https://github.com/gin-gonic/gin)

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 

[中文文档](README_cn.md)

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
- func(*api.Context) //Custom context type
- func(*api.Context,req) //Custom context type,Request parameters with req
  func(*api.Context,req)
  func(*gin.Context,*req)
  func(*gin.Context,req)


### Sample code

   ```go
   
   package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
	"github.com/xxjwxc/ginrpc/base/api"
)

type ReqTest struct {
	Access_token string `json:"access_token"`                 //access_token
	UserName     string `json:"user_name" binding:"required"` 
	Password     string `json:"password"`                     
}

// TestFun1 go-gin default function callback address
func TestFun1(c *gin.Context) {
	fmt.Println(c.Params)
	c.String(200, "ok")
}

// TestFun2 Customize the function callback address of context
func TestFun2(c *api.Context) {
	fmt.Println(c.Params)
	c.JSON(http.StatusOK, "ok")
}

// TestFun3 Callback with custom context and parsed req parameters
func TestFun3(c *api.Context, req *ReqTest) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// TestFun4 Callback with go-gin context and parsed req parameters
func TestFun4(c *gin.Context, req ReqTest) {
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}

func main() {
	router := gin.Default()
	router.POST("/test1", base.GetHandlerFunc(TestFun1))
	router.POST("/test2", base.GetHandlerFunc(TestFun2))
	router.POST("/test3", base.GetHandlerFunc(TestFun3))
	router.POST("/test4", base.GetHandlerFunc(TestFun4))

	router.Run(":8080")
}

   
   
   ```

 - curl
  ```
  curl 'http://127.0.0.1:8080/test4' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```
