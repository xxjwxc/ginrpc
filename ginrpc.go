package ginrpc

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
	"github.com/xxjwxc/public/errors"
)

// _Base base struct
type _Base struct {
	// tag     int
	apiFun  NewAPIFunc
	apiType reflect.Type
	router  *gin.Engine
	prePath string
}

// Default new op obj
func Default() *_Base {
	b := new(_Base)
	b.Model(api.NewAPIFunc)

	return b
}

// New new customized base
func New(middleware NewAPIFunc) *_Base {
	b := new(_Base)
	b.Model(middleware)

	return b
}

// Model use custom context
func (b *_Base) Model(middleware NewAPIFunc) *_Base {
	if middleware == nil { // default middleware
		middleware = api.NewAPIFunc
	}

	b.apiFun = middleware // save callback

	rt := reflect.TypeOf(middleware(&gin.Context{}))
	if rt == nil || rt.Kind() != reflect.Ptr {
		panic("need pointer")
	}
	b.apiType = rt

	return b
}

// Group creates a new router group. You should add all the routes that have common middlewares or the same path prefix.
// For example, all the routes that use a common middleware for authorization could be grouped.
// Last : you can us gin.Group replace this also (添加路由前缀,也可以调用gin.Group来设置)
func (b *_Base) Group(prepath string) *_Base {
	b.prePath = prepath
	return b
}

// Register Registered by struct object,[prepath + bojname.]
func (b *_Base) Register(router *gin.Engine, cList ...interface{}) error {
	modPkg, modFile := getModuleInfo()
	fmt.Println(modPkg, modFile)

	for _, c := range cList {
		reflectVal := reflect.ValueOf(c)
		t := reflect.Indirect(reflectVal).Type()
		objPkg := t.PkgPath()
		objName := t.Name()
		fmt.Println(objPkg, objName)

		// find path
		objFile := evalSymlinks(modPkg, modFile, objPkg)
		fmt.Println(objFile)

		astPkgs, b := getAstPkgs(objPkg, objFile, objName)
		ast.Print(token.NewFileSet(), astPkgs)
		fmt.Println(b)

		typ := reflect.TypeOf(c)
		for m := 0; m < typ.NumMethod(); m++ {
			fmt.Println(typ.Method(m))
			fmt.Println(typ.Method(m).PkgPath)
		}
	}

	return nil
}

// RegisterHandlerFunc Multiple registration methods.获取并过滤要绑定的参数
func (b *_Base) RegisterHandlerFunc(router *gin.Engine, httpMethod []string, relativePath string, handlerFuncs ...interface{}) error {
	list := make([]gin.HandlerFunc, 0, len(handlerFuncs))
	for _, call := range handlerFuncs {
		list = append(list, b.HandlerFunc(call))
	}

	for _, v := range httpMethod {
		// method := strings.ToUpper(v)
		// switch method{
		// case "ANY":
		// 	router.Any(relativePath,list...)
		// default:
		// 	router.Handle(method,relativePath,list...)
		// }
		// or
		switch strings.ToUpper(v) {
		case "POST":
			router.POST(relativePath, list...)
		case "GET":
			router.GET(relativePath, list...)
		case "DELETE":
			router.DELETE(relativePath, list...)
		case "PATCH":
			router.PATCH(relativePath, list...)
		case "PUT":
			router.PUT(relativePath, list...)
		case "OPTIONS":
			router.OPTIONS(relativePath, list...)
		case "HEAD":
			router.HEAD(relativePath, list...)
		case "ANY":
			router.Any(relativePath, list...)
		default:
			return errors.Errorf("method:[%v] not support", httpMethod)
		}
	}

	return nil
}

// HandlerFunc Get and filter the parameters to be bound
func (b *_Base) HandlerFunc(handlerFunc interface{}) gin.HandlerFunc { // 获取并过滤要绑定的参数
	typ := reflect.ValueOf(handlerFunc).Type()
	if typ.NumIn() == 1 { // Parameter checking 参数检查
		ctxType := typ.In(0)

		// go-gin default method
		if ctxType == reflect.TypeOf(&gin.Context{}) {
			return handlerFunc.(func(*gin.Context))
		}

		// Customized context . 自定义的context
		if ctxType == b.apiType {
			method := reflect.ValueOf(handlerFunc)
			return func(c *gin.Context) {
				method.Call([]reflect.Value{reflect.ValueOf(b.apiFun(c))})
			}
		}
	}

	// Custom context type with request parameters .自定义的context类型,带request 请求参数
	call, err := b.getCallFunc3(handlerFunc)
	if err != nil { // Direct reporting error.
		panic(err)
	}

	return call
}

// CheckHandlerFunc Judge whether to match rules
func (b *_Base) CheckHandlerFunc(handlerFunc interface{}) bool { // 判断是否匹配规则
	typ := reflect.ValueOf(handlerFunc).Type()
	if typ.NumIn() == 1 || typ.NumIn() == 2 { // Parameter checking 参数检查
		ctxType := typ.In(0)

		// go-gin default method
		if ctxType == reflect.TypeOf(&gin.Context{}) {
			return true
		}

		// Customized context . 自定义的context
		if ctxType == b.apiType {
			return true
		}
	}
	return false
}
