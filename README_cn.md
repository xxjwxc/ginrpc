## golang gin 参数自动绑定工具
- 支持rpc自动映射
- 支持对象注册
- 支持注解路由
- 基于 [go-gin](https://github.com/gin-gonic/gin) 的 json restful 风格的golang基础库
- 自带请求参数过滤及绑定实现 binding:"required"  [validator](go-playground/validator.v8)
- 代码注册简单且支持多种注册方式


## api接口说明

### 支持3种接口模式

- func(*gin.Context) //gogin 原始接口
- func(*api.Context) //自定义的context类型
- func(*api.Context,req) //自定义的context类型,带request 请求参数
  func(*api.Context,req)
  func(*gin.Context,*req)
  func(*gin.Context,req)


### 示例代码

## 初始化项目(本项目以ginweb 为名字)
	``` go mod init ginweb ```

### 代码 (详细地址：)
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
