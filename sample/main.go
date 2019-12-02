package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

type Hello struct {
}

// @router /block [post]
func (s *Hello) HelloS(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

func (s *Hello) HelloS2(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

func (s *Hello) helloS3(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// ReqTest test req
type ReqTest1 struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

type RespTest1 struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

// testFun1 Default function callback address on gin
func testFun1(c *gin.Context) { // gin 默认的函数回调地址
	fmt.Println(c.Params)
	c.String(200, "ok")
}

// testFun2 Customize the function callback address of context
func testFun2(c *api.Context) { // 自定义context的函数回调地址
	fmt.Println(c.Params)
	c.JSON(http.StatusOK, "ok")
}

// testFun3 Callback with custom context and parsed req parameters
func testFun3(c *api.Context, req *ReqTest1) { // 带自定义context跟已解析的req参数回调方式
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// testFun4 Callback with go-gin context and parsed req parameters
func testFun4(c *gin.Context, req ReqTest1) { // 带默认context跟已解析的req参数回调方式
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}

func main() {
	base := ginrpc.New(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	})

	router := gin.Default()
	base.Register(router, new(Hello)) //, new(api.Hello))
}
