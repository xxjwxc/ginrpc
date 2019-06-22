package base

import (
	"errors"
	"net/http"
	"reflect"
	"runtime"

	"github.com/xie1xiao1jun/ginrpc/base/api"

	"github.com/gin-gonic/gin"
)

/*
 说明：支持3种类型的接口
 func(*gin.Context) //gogin 原始接口
 func(*Context) //自定义的context类型
 func(*Context,req) //自定义的context类型,带request 请求参数
*/
// type handlerFunc1 func(*gin.Context)
// type handlerFunc2 func(*api.Context)
// type handlerFunc3 func(*api.Context, interface{})

func _fun1(*gin.Context)              {}
func _fun2(*api.Context)              {}
func _fun3(*api.Context, interface{}) {}

// var fun1 handlerFunc1
// var fun2 handlerFunc2
// var fun3 handlerFunc3

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

		// 带 struct 定义
		// method := reflect.TypeOf(handlerFunc).Method(0)
		// return func(c *gin.Context) {
		// 	method.Func.Call([]reflect.Value{reflect.ValueOf(api.Newctx(c))})
		// }
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

// func getCallFunc3(handlerFunc interface{}) (func(*gin.Context), error) {
// 	typ := reflect.TypeOf(handlerFunc).Method(0)
// 	mt := typ.Type
// 	if mt.NumIn() != 2 { //参数检查
// 		return nil, errors.New("method " + reflect.ValueOf(handlerFunc).Type().Name() + " not support!")
// 	}

// 	var ctxType, reqType reflect.Type
// 	ctxType = mt.In(0)
// 	reqType = mt.In(1)
// 	if reflect.TypeOf(ctxType) != reflect.TypeOf(&gin.Context{}) &&
// 		reflect.TypeOf(ctxType) != reflect.TypeOf(&api.Context{}) {
// 		return nil, errors.New("method " + reflect.ValueOf(handlerFunc).Type().Name() + " first parm not support!")
// 	}

// 	reqIsValue := false //
// 	if reqType.Kind() == reflect.Ptr {
// 		reqIsValue = true
// 	} else {
// 		reqIsValue = false
// 	}
// 	method := reflect.TypeOf(handlerFunc).Method(0)
// 	return func(c *gin.Context) {
// 		var req reflect.Value
// 		if reqIsValue {
// 			req = reflect.New(reqType.Elem())
// 		} else {
// 			req = reflect.New(reqType)
// 		}

// 		if err := unmarshal(c, req.Interface()); err != nil { //返回错误信息
// 			c.JSON(200, message.GetErrorMsg(message.ParameterInvalid, err))
// 			return
// 		}

// 		if reqIsValue {
// 			req = req.Elem()
// 		}

// 		method.Func.Call([]reflect.Value{reflect.ValueOf(api.Newctx(c)), req})
// 	}, nil
// }

func unmarshal(c *gin.Context, v interface{}) error {
	return c.ShouldBind(v)
}
