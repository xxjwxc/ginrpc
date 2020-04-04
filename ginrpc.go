package ginrpc

import (
	"reflect"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
	"github.com/xxjwxc/public/errors"
)

// _Base base struct
type _Base struct {
	// tag     int
	isBigCamel bool // big camel style.大驼峰命名规则
	isDev      bool // if is development
	apiFun     NewAPIFunc
	apiType    reflect.Type
	router     *gin.Engine
	groupPath  string // group path.添加路由前缀
	outPath    string // output path.输出目录
}

// Option overrides behavior of Connect.
type Option interface {
	apply(*_Base)
}

type optionFunc func(*_Base)

func (f optionFunc) apply(o *_Base) {
	f(o)
}

// WithOutPath set output path dir with router.go file.设置输出目录
func WithOutPath(path string) Option {
	return optionFunc(func(o *_Base) {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		o.outPath = path
	})
}

// WithCtx use custom context.设置自定义context
func WithCtx(middleware NewAPIFunc) Option {
	return optionFunc(func(o *_Base) {
		o.Model(middleware)
	})
}

// WithDebug set build is development.设置debug模式(默认debug模式)
func WithDebug(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.Dev(b)
	})
}

// WithBigCamel set build is BigCamel.是否大驼峰模式
func WithBigCamel(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.isBigCamel = b
	})
}

// WithGroup Add route prefix.添加路由前缀
func WithGroup(prepath string) Option {
	return optionFunc(func(o *_Base) {
		o.Group(prepath)
	})
}

// Default new op obj
func Default() *_Base {
	b := new(_Base)
	b.Model(api.NewAPIFunc)
	b.Dev(true)

	return b
}

// New new customized base
func New(opts ...Option) *_Base {
	b := Default() // default option

	for _, o := range opts {
		o.apply(b)
	}

	return b
}

// Dev set build is development
func (b *_Base) Dev(isDev bool) {
	b.isDev = isDev
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
	prepath = strings.Replace(prepath, "\\", "/", -1)
	if !strings.HasSuffix(prepath, "/") {
		prepath += "/"
	}
	b.groupPath = prepath
	return b
}

// Register Registered by struct object,[prepath + bojname.]
func (b *_Base) Register(router *gin.Engine, cList ...interface{}) bool {
	if b.isDev {
		b.tryGenRegister(router, cList...)
	}

	return b.register(router, cList...)
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

		panic("method " + runtime.FuncForPC(reflect.ValueOf(handlerFunc).Pointer()).Name() + " not support!")
	}

	// Custom context type with request parameters .自定义的context类型,带request 请求参数
	call, err := b.getCallFunc3(reflect.ValueOf(handlerFunc))
	if err != nil { // Direct reporting error.
		panic(err)
	}

	return call
}

// CheckHandlerFunc Judge whether to match rules
func (b *_Base) CheckHandlerFunc(handlerFunc interface{}) (int, bool) { // 判断是否匹配规则,返回参数个数
	typ := reflect.ValueOf(handlerFunc).Type()
	return b.checkHandlerFunc(typ, false)
}
