package ginrpc

import (
	"reflect"

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

// parmInfo 参数类型描述
type parmInfo struct {
	Pkg    string // 包名
	Type   string // 类型
	Import string // import 包
}

// store the comment for the controller method. 生成注解路由
type genComment struct {
	RouterPath string
	Note       string // 注释
	Methods    []string
}

// router style list.路由规则列表
type genRouterInfo struct {
	GenComment  genComment
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

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()

	genTemp = `
	package {{.PkgName}}
	
	import (
		"github.com/xxjwxc/ginrpc"
	)
	
	func init() {
		ginrpc.SetVersion({{.Tm}})
		{{range .List}}ginrpc.AddGenOne("{{.HandFunName}}", "{{.GenComment.RouterPath}}", []string{ {{GetStringList .GenComment.Methods}} })
		{{end}} }
	`
)
