package main

import (
	"fmt"
	"net/http"

	_ "ginweb/routers" // debug模式需要添加[mod]/routers 注册注解路由

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

// ReqTest demo struct
type ReqTest struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
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

//Hello3 带自定义context跟已解析的req参数回调方式,err,resp 返回模式
func (s *Hello) Hello3(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	//c.JSON(http.StatusOK, req)
	return &req, nil
}

//TestFun1 gin 默认的函数回调地址
func TestFun1(c *gin.Context) {
	c.String(200, "ok")
}

//TestFun2 自定义context的函数回调地址
func TestFun2(c *api.Context) {
	c.JSON(http.StatusOK, "ok")
}

//TestFun3 带自定义context跟已解析的req参数回调方式
func TestFun3(c *api.Context, req *ReqTest) {
	fmt.Println(req)
	c.WriteJSON(req)
}

//TestFun4 带自定义context跟已解析的req参数回调方式
func TestFun4(c *gin.Context, req ReqTest) {
	fmt.Println(req)
	c.JSON(http.StatusOK, req)
}

//TestFun5 带自定义context跟已解析的req参数回调方式,err,resp 返回模式
func TestFun5(c *gin.Context, req ReqTest) (ReqTest, error) {
	fmt.Println(req)
	//c.JSON(http.StatusOK, req)
	return req, nil
}

//TestFun6 带自定义context跟已解析的req参数回调方式,err,resp 返回模式
func TestFun6(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	//c.JSON(http.StatusOK, req)
	return &req, nil
}

func main() {
	base := ginrpc.New(ginrpc.WithCtx(func(c *gin.Context) interface{} {
		return api.NewCtx(c)
	}), ginrpc.WithDebug(true), ginrpc.WithGroup("xxjwxc"))
	router := gin.Default()

	router.POST("/test5", base.HandlerFunc(TestFun5)) // 函数注册
	router.POST("/test6", base.HandlerFunc(TestFun6)) // 函数注册

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
