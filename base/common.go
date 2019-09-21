package base

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/xxjwxc/public/errors"

	"github.com/xxjwxc/ginrpc/base/api"

	"github.com/gin-gonic/gin"
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

// GetHandlerFunc Get and filter the parameters to be bound
func GetHandlerFunc(handlerFunc interface{}) gin.HandlerFunc { // 获取并过滤要绑定的参数
	// go-gin default method
	if reflect.TypeOf(handlerFunc) == reflect.TypeOf(_fun1) {
		return handlerFunc.(func(*gin.Context))
	}

	// Customized context . 自定义的context
	if reflect.TypeOf(handlerFunc) == reflect.TypeOf(_fun2) {
		method := reflect.ValueOf(handlerFunc)
		return func(c *gin.Context) {
			method.Call([]reflect.Value{reflect.ValueOf(api.NewCtx(c))})
		}
	}

	// Custom context type with request parameters .自定义的context类型,带request 请求参数
	call, err := getCallFunc3(handlerFunc)

	if err != nil { // Direct reporting error.
		panic(err)
	}

	return call
}

// Custom context type with request parameters
func getCallFunc3(handlerFunc interface{}) (func(*gin.Context), error) {
	typ := reflect.ValueOf(handlerFunc).Type()
	if typ.NumIn() != 2 { // Parameter checking 参数检查
		return nil, errors.New("method " + runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name() + " not support!")
	}

	var ctxType, reqType reflect.Type
	ctxType = typ.In(0)
	reqType = typ.In(1)
	reqIsGinCtx := false
	if ctxType != reflect.TypeOf(&gin.Context{}) &&
		ctxType != reflect.TypeOf(&api.Context{}) {
		return nil, errors.New("method " + runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name() + " first parm not support!")
	}

	if ctxType == reflect.TypeOf(&gin.Context{}) {
		reqIsGinCtx = true
	}

	reqIsValue := true
	if reqType.Kind() == reflect.Ptr {
		reqIsValue = false
	}

	method := reflect.ValueOf(handlerFunc)
	return func(c *gin.Context) {
		var req reflect.Value
		if reqIsValue {
			req = reflect.New(reqType)
		} else {
			req = reflect.New(reqType.Elem())
		}

		if err := unmarshal(c, req.Interface()); err != nil { // Return error message.返回错误信息
			c.JSON(http.StatusBadRequest, gin.H{"state": false, "code": 1001, "error": err.Error()})
			return
		}

		if reqIsValue {
			req = req.Elem()
		}

		if reqIsGinCtx {
			method.Call([]reflect.Value{reflect.ValueOf(c), req})
		} else {
			method.Call([]reflect.Value{reflect.ValueOf(api.NewCtx(c)), req})
		}

	}, nil
}

func unmarshal(c *gin.Context, v interface{}) error {
	return c.ShouldBind(v)
}
