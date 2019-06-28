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
 说明：支持3种类型的接口
 func(*gin.Context) //gogin 原始接口
 func(*Context) //自定义的context类型
 func(*Context,req) //自定义的context类型,带request 请求参数
*/

func _fun1(*gin.Context)              {}
func _fun2(*api.Context)              {}
func _fun3(*api.Context, interface{}) {}

func GetHandlerFunc(handlerFunc interface{}) gin.HandlerFunc {
	//gin默认方法
	if reflect.TypeOf(handlerFunc) == reflect.TypeOf(_fun1) {
		return handlerFunc.(func(*gin.Context)) //可以添加func 包装调用前后
	}

	//自定义的context
	if reflect.TypeOf(handlerFunc) == reflect.TypeOf(_fun2) {
		method := reflect.ValueOf(handlerFunc)
		return func(c *gin.Context) {
			method.Call([]reflect.Value{reflect.ValueOf(api.Newctx(c))})
		}
	}

	//自定义的context类型,带request 请求参数
	call, err := getCallFunc3(handlerFunc)

	if err != nil { //直接
		panic(err)
	}

	return call
}

//
func getCallFunc3(handlerFunc interface{}) (func(*gin.Context), error) {
	typ := reflect.ValueOf(handlerFunc).Type()
	if typ.NumIn() != 2 { //参数检查
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

	reqIsValue := true //
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

		if err := unmarshal(c, req.Interface()); err != nil { //返回错误信息
			c.JSON(http.StatusBadRequest, gin.H{"state": false, "code": 1001, "error": err.Error()})
			return
		}

		if reqIsValue {
			req = req.Elem()
		}

		if reqIsGinCtx {
			method.Call([]reflect.Value{reflect.ValueOf(c), req})
		} else {
			method.Call([]reflect.Value{reflect.ValueOf(api.Newctx(c)), req})
		}

	}, nil
}

func unmarshal(c *gin.Context, v interface{}) error {
	return c.ShouldBind(v)
}
