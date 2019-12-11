# [English](README.md)

## [ginprc](https://github.com/xxjwxc/ginrpc) 注解路由，自动参数绑定工具

[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 

## golang gin 参数自动绑定工具
- 支持对象自动注册及注解路由
- 支持参数自动绑定
- 自带请求参数过滤及绑定实现 binding:"required"  [validator](go-playground/validator.v8)

### 支持3种接口模式

- func(*gin.Context) //go-gin 原始接口

  func(*api.Context) //自定义的context类型

- func(*api.Context,req) //自定义的context类型,带request 请求参数

- func(*gin.Context,*req) //go-gin context类型,带request 请求参数


## 1.参数自动绑定

```go

package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc"
	"github.com/xxjwxc/ginrpc/api"
)

type ReqTest struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
}

//TestFun4 带自定义context跟已解析的req参数回调方式
func TestFun4(c *api.Context, req ReqTest) {
	fmt.Println(req)
	c.WriteJSON(req) // 返回结果
}

func main() {
	base := ginrpc.New()
	router := gin.Default()
	router.POST("/test4", base.HandlerFunc(TestFun4))
	router.Run(":8080")
}

   ```

[更多>>](https://github.com/xxjwxc/ginrpc/tree/master/sample/ginweb)

- curl

  ```
  curl 'http://127.0.0.1:8080/test4' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'

  ```

## 2.对象注册(注解路由)

### 初始化项目(本项目以ginweb 为名字)

	``` go mod init ginweb ```

### 代码 

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
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
}

type Hello struct {
}

// Hello 带注解路由(参考beego形式)
// @router /block [post,get]
func (s *Hello) Hello(c *api.Context, req *ReqTest) {
	fmt.Println(req)
	c.WriteJSON(req) // 返回结果
}

// Hello2 不带注解路由(参数为2默认post)
func (s *Hello) Hello2(c *gin.Context, req ReqTest) {
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok") // gin 默认返回结果
}

func main() {
	base := ginrpc.New(ginrpc.WithGroup("xxjwxc"))
	router := gin.Default()
	base.Register(router, new(Hello)) // 对象注册 like(go-micro)
	router.Run(":8080")
}
   ```

[更多>>](https://github.com/xxjwxc/ginrpc/tree/master/sample/ginweb)

### 执行curl，可以自动参数绑定。直接看结果

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/block' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

  ```
  curl 'http://127.0.0.1:8080/xxjwxc/hello.hello2' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```

------------------------------------------------------

### -注解路由相关说明

```
 // @router /block [post,get]
@router 标记  /block 路由 [post,get] method 调用方式

 ```

 #### 说明:如果对象函数中不加注解路由，系统会默认添加注解路由。post方式：带req(2个参数(ctx,req))，get方式为一个参数(ctx)

### 1. 注解路由会自动创建[root]/routers/gen_router.go 文件 需要在调用时加：

	```
	_ "[mod]/routers" // debug模式需要添加[mod]/routers 注册注解路由
	```

	默认也会在项目根目录生成 `gen_router.data` 文件(保留此文件，可以不用添加上面代码嵌入)

### 2. 注册函数说明

	ginrpc.WithCtx ： 设置自定义context

	ginrpc.WithDebug(true) : 设置debug模式

	ginrpc.WithGroup("xxjwxc") : 添加路由前缀 (也可以使用gin.Group 分组)

	ginrpc.WithBigCamel(true) : 设置大驼峰标准(false 为web模式，_,小写)

[更多>>](https://godoc.org/github.com/xxjwxc/ginrpc)

### 2. 注解路由调用demo：[ginweb](/sample/ginweb)


## 下一步

	1.导出api文档

	2.导出postman测试配置

### 代码地址： [ginprc](https://github.com/xxjwxc/ginrpc) 如果喜欢请给星支持
