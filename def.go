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

// RecoverErrorFunc recover 错误设置
type RecoverErrorFunc func(interface{})

// Exporter 可选 opt-in 接口:实现该接口的 controller 即使没有
// 由 @Router 生成的路由表条目,也会在关闭 WithAutoRegister 时被注册。
// 用于不方便写注解、但又希望显式暴露方法的场景(方法仍需签名匹配)。
type Exporter interface {
	GinrpcExporter()
}

// parmInfo 参数类型描述
type parmInfo struct {
	Pkg    string // 包名
	Type   string // 类型
	Import string // import 包
}

// store the comment for the controller method. 生成注解路由
type genComment struct {
	RouterPath     string
	Note           string // 注释
	Methods        []string
	ThirdPartyList []GenThirdParty // 第三方注解路由
	Ignored        bool            // @Router - [-] 显式忽略标记,运行期 register 命中即跳过
}

type GenThirdParty struct {
	Name string
	Data string
	Note string // 注释
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
		{{range .List}}{{if .GenComment.Ignored}}ginrpc.AddGenIgnore("{{.HandFunName}}")
		{{else}}ginrpc.AddGenOne("{{.HandFunName}}", "{{.GenComment.RouterPath}}", []string{ {{GetStringList .GenComment.Methods}} },[]ginrpc.GenThirdParty{ {{GetPartyList .GenComment.ThirdPartyList}} },{{GetNote .GenComment.Note}})
		{{end}}{{end}} }
	`
)
