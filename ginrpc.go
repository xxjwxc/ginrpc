package ginrpc

import (
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"

	"github.com/xxjwxc/public/mylog"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
	"github.com/xxjwxc/public/errors"
	"github.com/xxjwxc/public/message"
)

// _Base base struct
type _Base struct {
	isBigCamel         bool // big camel style.大驼峰命名规则
	isDev              bool // if is development(默认 false,仅 WithDebug(true) 开启)
	apiFun             NewAPIFunc
	apiType            reflect.Type
	outPath            string // output path.输出目录
	beforeAfter        []GinBeforeAfter
	isOutDoc           bool
	recoverErrorFunc   RecoverErrorFunc
	autoRegister       bool              // 兼容开关:为 true 时恢复"注册所有签名匹配方法"的旧行为(默认 false)
	importFiles        map[string]string // WithImportFile 实例级收集,避免 Option 带全局副作用
	importFilesApplied bool              // 幂等保护:只同步一次到 myast 全局表
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

// WithDebug 开启/关闭开发模式(路由表生成)。默认关闭:
// New()/Default() 不再在每次 Register 时做全量 AST 扫描 + 写文件,
// 只有显式 WithDebug(true) 才会生成 routers/gen_router.go 与路由数据。
func WithDebug(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.Dev(b)
	})
}

// WithAutoRegister 兼容开关。默认 false(更安全):只注册显式标注
// (@Router 生成了路由表条目)或实现 Exporter 的方法,内部方法不会被暴露。
// 设为 true 恢复旧行为——所有签名匹配的导出方法(含无注解的)都会被注册,
// 缺省路由由参数个数推断。
func WithAutoRegister(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.autoRegister = b
	})
}

// WithCtxType 显式指定自定义上下文类型。Model() 不再在构造期调用用户回调,
// 默认上下文推断为 *api.Context;若 WithCtx 返回其它类型,用本 Option 指定。
func WithCtxType(t reflect.Type) Option {
	return optionFunc(func(o *_Base) {
		if t == nil {
			return
		}
		if t.Kind() != reflect.Ptr {
			panic("need pointer")
		}
		o.apiType = t
	})
}

// WithImportFile 添加自定义import文件列表。
// 历史上这里直接调用 myast.AddImportFile(k,v) 修改全局 map,
// 让一个"设置项"带上了进程级副作用。改为实例级收集,仅在实际做代码生成
// (tryGenRegister)时才同步到 myast 全局表(myast 只支持全局写入)。
func WithImportFile(k, v string) Option {
	return optionFunc(func(o *_Base) {
		if o.importFiles == nil {
			o.importFiles = make(map[string]string)
		}
		o.importFiles[k] = v
	})
}

// WithBigCamel set build is BigCamel.是否大驼峰模式
func WithBigCamel(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.isBigCamel = b
	})
}

// WithOutDoc set is out doc.是否输出文档
func WithOutDoc(b bool) Option {
	return optionFunc(func(o *_Base) {
		o.isOutDoc = b
	})
}

// WithBeforeAfter set before and after call.设置对象调用前后执行中间件
func WithBeforeAfter(beforeAfter GinBeforeAfter) Option {
	return optionFunc(func(o *_Base) {
		o.beforeAfter = append(o.beforeAfter, beforeAfter)
	})
}

// WithRecover 设置 recover 回调。
// 历史上 New() 在应用完所有 Option 之后会强制把 recover 改回默认实现,
// 导致调用方无法定制,这里改为通过 Option 设置。
func WithRecover(f RecoverErrorFunc) Option {
	return optionFunc(func(o *_Base) {
		if f != nil {
			o.recoverErrorFunc = f
		}
	})
}

// Default new op obj
func Default() *_Base {
	b := new(_Base)
	b.Model(api.NewAPIFunc)
	b.Dev(false) // 默认非开发模式:不再每次 Register 都做 AST 扫描 + 写文件
	b.SetRecover(func(err interface{}) {
		// 默认 recover 回调补上堆栈日志(历史实现只 mylog.Error(err))。
		// 框架侧 recoverHandler 会另外带 errorID 记录一份用于关联定位。
		mylog.Error(fmt.Sprintf("[recover] %v\n%s", err, debug.Stack()))
	})

	return b
}

// New new customized base
func New(opts ...Option) *_Base {
	b := Default() // default option

	for _, o := range opts {
		o.apply(b)
	}

	// 注意:不要在这里覆写 recoverErrorFunc,
	// 否则 WithRecover 传入的定制实现会被静默丢弃(历史 bug)。
	return b
}

// Dev set build is development
func (b *_Base) Dev(isDev bool) {
	b.isDev = isDev
}

// SetRecover set recover err call
func (b *_Base) SetRecover(errfun func(interface{})) {
	b.recoverErrorFunc = errfun
}

// OutDoc set if out doc. 设置是否输出接口文档
func (b *_Base) OutDoc(isOutDoc bool) {
	b.isOutDoc = isOutDoc
}

// Model use custom context
func (b *_Base) Model(middleware NewAPIFunc) *_Base {
	if middleware == nil { // default middleware
		middleware = api.NewAPIFunc
	}

	b.apiFun = middleware // save callback

	// 不再通过 middleware(&gin.Context{}) 在构造期探测类型:
	// 空上下文会让访问 c.Request 的自定义 apiFun 直接 panic,
	// 且 NewAPIFunc 的静态返回类型是 interface{},反射也看不到具体类型。
	// 默认自定义上下文推断为 *api.Context(覆盖 api.NewCtx 等绝大多数用法);
	// 若使用其它类型,请用 WithCtxType 显式指定(避免调用用户回调)。
	// 已由 WithCtxType 指定过则保留。
	if b.apiType == nil {
		b.apiType = reflect.TypeOf(&api.Context{})
	}

	return b
}

// Register Registered by struct object,[prepath + bojname.]
func (b *_Base) Register(router gin.IRoutes, cList ...interface{}) bool {
	if b.isDev {
		b.tryGenRegister(router, cList...)
	}

	return b.register(router, cList...)
}

// RegisterHandlerFunc Multiple registration methods.获取并过滤要绑定的参数
func (b *_Base) RegisterHandlerFunc(router gin.IRoutes, httpMethod []string, relativePath string, handlerFuncs ...interface{}) error {
	list := make([]gin.HandlerFunc, 0, len(handlerFuncs))
	for _, call := range handlerFuncs {
		list = append(list, b.HandlerFunc(call))
	}

	for _, v := range httpMethod {
		if err := handleHTTPMethod(router, v, relativePath, list...); err != nil {
			return err
		}
	}

	return nil
}

// handleHTTPMethod 把 HTTP 方法名映射到 gin 的注册函数。
// 抽出来避免 RegisterHandlerFunc 与 registerHandlerObj 各写一份 switch 逻辑不一致。
func handleHTTPMethod(router gin.IRoutes, method, relativePath string, handlers ...gin.HandlerFunc) error {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case "POST":
		router.POST(relativePath, handlers...)
	case "GET":
		router.GET(relativePath, handlers...)
	case "DELETE":
		router.DELETE(relativePath, handlers...)
	case "PATCH":
		router.PATCH(relativePath, handlers...)
	case "PUT":
		router.PUT(relativePath, handlers...)
	case "OPTIONS":
		router.OPTIONS(relativePath, handlers...)
	case "HEAD":
		router.HEAD(relativePath, handlers...)
	case "ANY":
		// 仅显式注解 [any] 才开放所有动词;默认路由已收窄为 GET/POST
		router.Any(relativePath, handlers...)
	case "":
		// 空方法名/占位符不能静默落到 router.Any(会开放 DELETE/PUT 等)
		return errors.Errorf("empty http method not support")
	default:
		return errors.Errorf("method:[%v] not support", method)
	}
	return nil
}

// panicSeq / newErrorID:为每次 panic 生成一个短关联 ID。
var panicSeq uint64

func newErrorID() string {
	return fmt.Sprintf("%x-%x", time.Now().UnixNano(), atomic.AddUint64(&panicSeq, 1))
}

// recoverHandler recover 时的统一处理:内部用 debug.Stack() 打日志并关联 errorID,
// 对外只给固定文案,绝不把 panic 内容(可能是 password/token)回显客户端。
func (b *_Base) recoverHandler(c *gin.Context, err interface{}) {
	errorID := newErrorID()

	// 框架内部日志:关联 errorID + 完整堆栈,仅服务端可见。
	mylog.Error(fmt.Sprintf("[panic][errorID=%s] %v\n%s", errorID, err, debug.Stack()))

	// 用户自定义 recover 回调(WithRecover / 默认实现),仅作额外观测,
	// 不影响对外响应。
	if b.recoverErrorFunc != nil {
		b.recoverErrorFunc(err)
	}

	if c == nil || c.Writer.Written() { // 已经写过响应就不重复写
		return
	}

	msg := message.GetErrorMsg(message.InValidOp)
	msg.Error = "internal server error" // 固定文案,不泄漏 panic 内容
	msg.Data = errorID                  // 客户端凭此 ID 关联内部日志
	c.AbortWithStatusJSON(http.StatusInternalServerError, msg)
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

		// 常见路径 func(*api.Context) + 默认 apiFun:类型断言预编译静态闭包,
		// 避免每请求 reflect.Value.Call。自定义 apiFun 走下面的反射路径,
		// 由 WithCtxType/Model 保证类型匹配。
		if f, ok := handlerFunc.(func(*api.Context)); ok &&
			b.apiType == ctxType &&
			reflect.ValueOf(b.apiFun).Pointer() == reflect.ValueOf(api.NewAPIFunc).Pointer() {
			return func(c *gin.Context) {
				defer func() {
					if err := recover(); err != nil {
						b.recoverHandler(c, err)
					}
				}()
				f(b.apiFun(c).(*api.Context))
			}
		}

		// Customized context . 自定义的context
		if ctxType == b.apiType {
			method := reflect.ValueOf(handlerFunc)
			return func(c *gin.Context) {
				defer func() {
					if err := recover(); err != nil {
						b.recoverHandler(c, err)
					}
				}()
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
