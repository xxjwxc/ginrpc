package main

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
	"github.com/xxjwxc/ginrpc/base/api"
)

//ReqTest .
type ReqTest1 struct {
	AccessToken string `json:"access_token"`                 //access_token
	UserName    string `json:"user_name" binding:"required"` //用户名
	Password    string `json:"password"`                     //新密码
}

//TestFun1 gin 默认的函数回调地址
func TFun1(c *gin.Context) {
	fmt.Println(c.Params)
	c.String(200, "ok")
}

//TestFun2 自定义context的函数回调地址
func TFun2(c *api.Context) {
	fmt.Println(c.Params)
	c.JSON(http.StatusOK, "ok")
}

//TestFun3 带自定义context跟已解析的req参数回调方式
func TFun3(c *api.Context, req *ReqTest) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

//TestFun4 带自定义context跟已解析的req参数回调方式
func TFun4(c *gin.Context, req ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}

func TestMain(t *testing.T) {
	router := gin.Default()
	router.POST("/test1", base.GetHandlerFunc(TFun1))
	router.POST("/test2", base.GetHandlerFunc(TFun2))
	router.POST("/test3", base.GetHandlerFunc(TFun3))
	router.POST("/test4", base.GetHandlerFunc(TFun4))

	//router.Run(":8080")
}
