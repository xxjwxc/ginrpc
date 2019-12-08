package ginrpc

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
)

func TestModelObj(t *testing.T) {
	base := New(WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}))

	router := gin.Default()
	base.Register(router, "/", new(Hello))
}

func TestModelFunc(t *testing.T) {
	// base := Default()
	// base.Model(func(c *gin.Context) interface{} {
	// 	return api.NewCtx(c)
	// })
	base := New(WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}))

	router := gin.Default()
	base.RegisterHandlerFunc(router, []string{"post", "get"}, "/test", testFun1)
	router.POST("/test1", base.HandlerFunc(testFun1))
	router.POST("/test2", base.HandlerFunc(testFun2))
	router.POST("/test3", base.HandlerFunc(testFun3))
	router.POST("/test4", base.HandlerFunc(testFun4))

	// router.Run(":8080")
}

// ReqTest test req
type ReqTest1 struct {
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

//////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////

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
func (s *Hello) HelloS2(c *api.Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	fmt.Println(s.Index)
	c.JSON(http.StatusOK, "ok")
}
func TestObj(t *testing.T) {
	base := New(WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), WithDebug(true), WithGroup("xxjwxc"), WithBigCamel(true))

	router := gin.Default()
	h := new(Hello)
	h.Index = 123
	base.Register(router, h) //, new(api.Hello))
	// router.Run(":8080")
}
