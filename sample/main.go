package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

// ReqTest struct test
type ReqTest struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
}

//TestFun4 带自定义context跟已解析的req参数回调方式
func TestFun4(c *api.Context, req ReqTest) {
	fmt.Println(req)
	c.WriteJSON(req)
}

func main() {
	base := ginrpc.New()
	router := gin.Default()
	router.POST("/test4", base.HandlerFunc(TestFun4))
	router.Run(":8080")
}
