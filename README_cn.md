[![Build Status](https://travis-ci.org/xxjwxc/ginrpc.svg?branch=master)](https://travis-ci.org/xxjwxc/ginrpc)
[![Go Report Card](https://goreportcard.com/badge/github.com/xxjwxc/ginrpc)](https://goreportcard.com/report/github.com/xxjwxc/ginrpc)
[![codecov](https://codecov.io/gh/xxjwxc/ginrpc/branch/master/graph/badge.svg)](https://codecov.io/gh/xxjwxc/ginrpc)
[![GoDoc](https://godoc.org/github.com/xxjwxc/ginrpc?status.svg)](https://godoc.org/github.com/xxjwxc/ginrpc)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go) 
# [English](README.md)

## [ginprc](https://github.com/xxjwxc/ginrpc) 注解路由，自动参数绑定工具

![img](/image/ginrpc.gif)

## doc 

![doc](/image/ginrpc_doc.gif)


## golang gin 参数自动绑定工具
- 支持对象自动注册及注解路由
- 支持参数自动绑定
- 自带请求参数过滤及绑定实现 binding:"required"  [validator](go-playground/validator.v8)
- 支持 [grpc-go](https://github.com/grpc/grpc-go) 绑定模式
- 支持[swagger 文档](http://editor.swagger.io/)导出 [MORE](https://github.com/gmsec/gmsec)
- 支持[markdown/mindoc 文档](https://www.iminho.me/)导出 [MORE](https://github.com/gmsec/gmsec)

- [更多请看](https://github.com/gmsec/gmsec)

## 安装使用
- go mod:
```
go get -u github.com/xxjwxc/ginrpc@master
```

### 支持多种接口模式

- func(*gin.Context) //go-gin 原始接口

  func(*api.Context) //自定义的context类型

- func(*api.Context,req) //自定义的context类型,带request 请求参数

- func(*gin.Context,*req) //go-gin context类型,带request 请求参数

- func(*gin.Context,*req)(*resp,error) //go-gin context类型,带request 请求参数,带错误返回参数 ==> [grpc-go](https://github.com/grpc/grpc-go)

   func(*gin.Context,req)(resp,error)

## 一. 参数自动绑定/对象注册(注解路由)

### 初始化项目(本项目以gmsec 为名字)

` go mod init gmsec `

### 代码 

```go
package main

import (
	"fmt"
	"net/http"

	_ "gmsec/routers" // debug模式需要添加[mod]/routers 注册注解路由
	"github.com/xxjwxc/public/mydoc/myswagger" // swagger 支持
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
// @Router /block [post,get]
func (s *Hello) Hello(c *api.Context, req *ReqTest) {
	fmt.Println(req)
	c.WriteJSON(req) // 返回结果
}

// Hello2 不带注解路由(参数为2默认post)
func (s *Hello) Hello2(c *gin.Context, req ReqTest) {
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok") // gin 默认返回结果
}

// Hello3 [grpc-go](https://github.com/grpc/grpc-go) 模式
func (s *Hello) Hello3(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	return &req,nil
}

//TestFun6 带自定义context跟已解析的req参数回调方式,err,resp 返回模式
func TestFun6(c *gin.Context, req ReqTest) (*ReqTest, error) {
	fmt.Println(req)
	//c.JSON(http.StatusOK, req)
	return &req, nil
}

func main() {
	// swagger
	myswagger.SetHost("https://localhost:8080")
	myswagger.SetBasePath("gmsec")
	myswagger.SetSchemes(true, false)
	// -----end --
	base := ginrpc.New(ginrpc.WithGroup("xxjwxc"))
	router := gin.Default()
	base.Register(router, new(Hello)) // 对象注册 like(go-micro)
	router.POST("/test6", base.HandlerFunc(TestFun6))                            // 函数注册
	base.RegisterHandlerFunc(router, []string{"post", "get"}, "/test", TestFun6) // 多种请求方式注册
	router.Run(":8080")
}
   ```

[更多>>](https://github.com/gmsec/gmsec)

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
// @Router /block [post,get]
@Router 标记 
/block 路由 
[post,get] method 调用方式

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

### 2. 注解路由调用demo：[gmsec](https://github.com/gmsec/gmsec)

### 3. 支持绑定grpc函数: [gmsec](https://github.com/gmsec/gmsec)

## 二. swagger/markdown/mindoc 文档生成说明

### 1.对于对象注册`ginrpc.Register`模式,支持文档导出
### 2.导出支持注解路由,支持参数注释,支持默认值(`tag`.`default`)
### 3.默认导出路径:(`/docs/swagger/swagger.json`,`/docs/markdown`)
### 4 struct demo
```
type ReqTest struct {
	AccessToken string `json:"access_token"`
	UserName    string `json:"user_name" binding:"required"` // 带校验方式
	Password    string `json:"password"`
}
```

- [更多 >>>](https://github.com/xxjwxc/gmsec)


## Stargazers over time

[![Stargazers over time](https://starchart.cc/xxjwxc/ginrpc.svg)](https://starchart.cc/xxjwxc/ginrpc)

## 下一步

- 添加服务发现

### 代码地址： [ginprc](https://github.com/xxjwxc/ginrpc) 如果喜欢请给星支持
