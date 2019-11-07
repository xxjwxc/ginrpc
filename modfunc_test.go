package ginrpc

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
	"github.com/xxjwxc/ginrpc/base/api"
)

func TestModelFunc(t *testing.T) {
	base := base.Default()
	base.Model(&api.Context{}).NewCustomCtxCall(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	})

	router := gin.Default()
	router.POST("/test1", base.GetHandlerFunc(testFun1))
	router.POST("/test2", base.GetHandlerFunc(testFun2))
	router.POST("/test3", base.GetHandlerFunc(testFun3))
	router.POST("/test4", base.GetHandlerFunc(testFun4))

	router.Run(":8080")
}

// ReqTest test req
type ReqTest struct {
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
func testFun3(c *api.Context, req *ReqTest) { // 带自定义context跟已解析的req参数回调方式
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// testFun4 Callback with go-gin context and parsed req parameters
func testFun4(c *gin.Context, req ReqTest) { // 带默认context跟已解析的req参数回调方式
	fmt.Println(c.Params)
	fmt.Println(req)

	c.JSON(http.StatusOK, req)
}
