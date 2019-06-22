## golang gin 参数自动绑定工具
- 基于 [go-gin](https://github.com/gin-gonic/gin) 的 json restful 风格的golang基础库
- 自带请求参数过滤及绑定实现
- 代码注册简单且支持多种注册方式

1、 目录结构说明

- ginrpc/base/common.go 基础库
- ginrpc/base/api/context.go 自定义context内容
- 支持参数自动检测 binding:"required"  [validator](go-playground/validator.v8)
- 支持rpc自动映射

2、api接口说明

### 支持3种接口模式

- func(*gin.Context) //gogin 原始接口
- func(*api.Context) //自定义的context类型
- func(*api.Context,req) //自定义的context类型,带request 请求参数
     func(*gin.Context,*req)
     ...... 等接口模式


### 示例代码

   ```go
   type ReqTest struct {
        Access_token string `json:"access_token"`                 //access_token
        UserName     string `json:"user_name" binding:"required"` //用户名
        Password     string `json:"password"`                     //新密码
    }

    //TestFun1 gin 默认的函数回调地址
    func TestFun1(c *gin.Context) {
    }

    //TestFun2 自定义context的函数回调地址
    func TestFun2(c *api.Context) {
    }

    //TestFun3 带自定义context跟已解析的req参数回调方式
    func TestFun3(c *api.Context, req *ReqTest) {
        fmt.Println(req)
    }

    //TestFun3 带自定义context跟已解析的req参数回调方式
    func TestFun4(c *gin.Context, req ReqTest) {
        fmt.Println(req)
    }

    func main() {
        router := gin.Default()
        router.POST("/test1", base.GetHandlerFunc(TestFun1))
        router.POST("/test2", base.GetHandlerFunc(TestFun2))
        router.POST("/test3", base.GetHandlerFunc(TestFun3))
        router.POST("/test4", base.GetHandlerFunc(TestFun4))

        router.Run(":8080")
    }
   ```

- curl
  ```
  curl 'http://127.0.0.1:8080/test4' -H 'Content-Type: application/json' -d '{"access_token":"111", "user_name":"222", "password":"333"}'
  ```
