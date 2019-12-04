package ginrpc

import (
	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
)

/*
 Description: Support three types of interfaces
 func(*gin.Context) go-gin raw interface
 func(*Context)  Custom context type
 func(*Context,req)  Custom context type with request request request parameters
*/

func _fun1(*gin.Context)              {}
func _fun2(*api.Context)              {}
func _fun3(*api.Context, interface{}) {}

// NewAPIFunc Custom context support
type NewAPIFunc func(*gin.Context) interface{}

// store the comment for the controller method. 生成注解路由
type genComment struct {
	RouterPath string
	Methods    []string
}

// router style list.路由规则列表
type genRouterInfo struct {
	genComment
	HandFunName string
}

type genInfo struct {
	List []genRouterInfo
	Tm   int64 //genout time
}

// // router style list.路由规则列表
// type genRouterList struct {
// 	list []genRouterInfo
// }

var genTemp string = `
package {{.PkgName}}

import (
	"github.com/xxjwxc/ginrpc"
)

func init() {
	ginrpc.SetVersion({{.Tm}})
	{{range .List}}ginrpc.AddGenOne("{{.HandFunName}}", "{{.RouterPath}}", []string{ {{GetStringList .Methods}} })
	{{end}} }
`
