package base

import (
	"reflect"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base/api"
	"github.com/xxjwxc/public/errors"
)

// base
type base struct {
	tag     int
	apiFun  NewAPIFunc
	apiType reflect.Type
}

// Default new op obj
func Default() *base {
	b := new(base)
	b.apiFun = api.NewAPIFunc
	b.apiType = reflect.TypeOf(api.Context{})

	return b
}

// GetHandlerFunc Get and filter the parameters to be bound
func (b *base) GetHandlerFunc(handlerFunc interface{}) gin.HandlerFunc { // 获取并过滤要绑定的参数
	if !b.checkTag() {
		panic(errors.New("method:Model and UseCustomCtx must use together"))
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

// CheckHandlerFunc Judge whether to match rules
func (b *base) CheckHandlerFunc(handlerFunc interface{}) bool { // 判断是否匹配规则
	if reflect.TypeOf(handlerFunc) == reflect.TypeOf(_fun1) {
		return true
	}

	return false
}

// NewCustomCtx use custom context
func (b *base) NewCustomCtxCall(middleware NewAPIFunc) *base { // 使用自定义 context
	b.apiFun = middleware
	b.tagOn(2)
	return b
}

// Model use custom model
func (b *base) Model(ty interface{}) *base {
	rt := reflect.TypeOf(ty)
	if rt == nil || rt.Kind() != reflect.Ptr {
		panic("need pointer")
	}
	b.tagOn(1)
	b.apiType = rt

	return b
}
