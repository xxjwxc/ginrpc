package main

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
	_ "github.com/xxjwxc/ginrpc/routers"
)

// Hello ...
type Hello struct {
	Index int
}

// HelloS ...
// @router /block [post,get]
func (s *Hello) HelloS(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// HelloS2 ...
// @router /block1 [post,get]
func (s *Hello) HelloS2(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	fmt.Println(s.Index)
	c.JSON(http.StatusOK, "ok")
}

// @router /block2 [post,get]
func (s *Hello) helloS3(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// ReqTest1 test req
type ReqTest1 struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

// RespTest1 test req
type RespTest1 struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

func main() {
	base := ginrpc.New(ginrpc.WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), ginrpc.WithDebug(true), ginrpc.WithGroup("xxjwxc"))

	router := gin.Default()
	h := new(Hello)
	h.Index = 123
	base.Register(router, h)
	router.Run(":8080")
}
