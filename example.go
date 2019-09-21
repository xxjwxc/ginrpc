package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
	"github.com/xxjwxc/ginrpc/base/api"
)

// ReqTest test req
type ReqTest struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

// TestFun1 Default function callback address on gin
func TestFun1(c *gin.Context) { // gin 默认的函数回调地址
	fmt.Println(c.Params)
	c.String(200, "ok")
}

// TestFun2 Customize the function callback address of context
func TestFun2(c *api.Context) { // 自定义context的函数回调地址
	fmt.Println(c.Params)
	c.JSON(http.StatusOK, "ok")
}

// TestFun3 Callback with custom context and parsed req parameters
func TestFun3(c *api.Context, req *ReqTest) { // 带自定义context跟已解析的req参数回调方式
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// TestFun4 Callback with go-gin context and parsed req parameters
func TestFun4(c *gin.Context, req ReqTest) { // 带默认context跟已解析的req参数回调方式
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
