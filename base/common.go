package base

import (
	"net/http"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/public/errors"
)

// func (b *base) initAPI() {
// 	typ := reflect.ValueOf(b.iFunc3).Type()
// 	if typ.NumIn() != 2 { // Parameter checking 参数检查
// 		panic(errors.New("method " + runtime.FuncForPC(
// 			reflect.ValueOf(b.iFunc3).Pointer()).Name() + " not support!"))
// 	}
// 	b.apiFun = api.NewAPIFunc
// 	b.apiType = typ.In(0)
// }

// Custom context type with request parameters
func (b *base) getCallFunc3(handlerFunc interface{}) (func(*gin.Context), error) {
	typ := reflect.ValueOf(handlerFunc).Type()
	if typ.NumIn() != 2 { // Parameter checking 参数检查
		return nil, errors.New("method " + runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name() + " not support!")
	}

	var ctxType, reqType reflect.Type
	ctxType = typ.In(0)
	reqType = typ.In(1)
	reqIsGinCtx := false
	// ctxType != reflect.TypeOf(gin.Context{}) &&
	// ctxType != reflect.Indirect(reflect.ValueOf(b.iAPIType)).Type()
	if ctxType != reflect.TypeOf(&gin.Context{}) &&
		ctxType != b.apiType {
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

		if err := b.unmarshal(c, req.Interface()); err != nil { // Return error message.返回错误信息
			c.JSON(http.StatusBadRequest, gin.H{"state": false, "code": 1001, "error": err.Error()})
			return
		}

		if reqIsValue {
			req = req.Elem()
		}

		if reqIsGinCtx {
			method.Call([]reflect.Value{reflect.ValueOf(c), req})
		} else {
			method.Call([]reflect.Value{reflect.ValueOf(b.apiFun(c)), req})
		}

	}, nil
}

func (b *base) unmarshal(c *gin.Context, v interface{}) error {
	return c.ShouldBind(v)
}

func (b *base) tagOn(n int) {
	b.tag |= n
}

func (b *base) checkTag() bool {
	if (b.tag&1) == 1 || ((b.tag<<1)&1) == 1 {
		return (b.tag & 0x11) == 1
	}
	return true
}
