package ginrpc

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unsafe"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
	"github.com/xxjwxc/public/errors"
)

// _Base base struct
type _Base struct {
	tag     int
	apiFun  NewAPIFunc
	apiType reflect.Type
	router  *gin.Engine
}

// Default new op obj
func Default() *_Base {
	b := new(_Base)
	b.apiFun = api.NewAPIFunc
	b.apiType = reflect.TypeOf(&api.Context{})

	return b
}

// New new customized base
func New(ty interface{}, middleware NewAPIFunc) *_Base {
	b := new(_Base)
	b.Model(ty).NewCustomCtxCall(middleware)

	return b
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

func call() {
	fmt.Println(runtime.Caller(2))
}

// Register Registered by struct object
func (b *_Base) Register(router *gin.Engine, cList ...interface{}) error {
	call()

	for _, c := range cList {
		reflectVal := reflect.ValueOf(c)

		t := reflect.Indirect(reflectVal).Type()
		fmt.Println(runtime.FuncForPC((uintptr)(unsafe.Pointer(b))).Name())

		fmt.Println(t)
		typ := reflect.TypeOf(c)
		hdlr := reflect.ValueOf(c)
		name := reflect.Indirect(hdlr).Type().Name()
		vtyp := reflect.Indirect(hdlr).Type()
		fmt.Println(vtyp.PkgPath())
		fmt.Println(reflect.Indirect(reflect.ValueOf(b)).Type().PkgPath())

		for m := 0; m < typ.NumMethod(); m++ {
			fmt.Println(typ.Method(m))
			fmt.Println(typ.Method(m).PkgPath)
			fmt.Println(name)
		}
	}

	return nil
}

// RegisterHandlerFunc Multiple registration methods
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
	if !b.checkTag() {
		panic(errors.New("method:Model and NewCustomCtxCall must use together"))
	}

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

// NewCustomCtx use custom context
func (b *_Base) NewCustomCtxCall(middleware NewAPIFunc) *_Base { // 使用自定义 context
	b.apiFun = middleware
	b.tagOn(2)
	return b
}

// Model use custom model
func (b *_Base) Model(ty interface{}) *_Base {
	rt := reflect.TypeOf(ty)
	if rt == nil || rt.Kind() != reflect.Ptr {
		panic("need pointer")
	}
	b.tagOn(1)
	b.apiType = rt

	return b
}
