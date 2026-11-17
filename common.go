package ginrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"net/http"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/xxjwxc/public/mylog"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/xxjwxc/public/errors"
	"github.com/xxjwxc/public/message"
	"github.com/xxjwxc/public/myast"
	"github.com/xxjwxc/public/mybigcamel"
	"github.com/xxjwxc/public/mydoc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// checkHandlerFunc Judge whether to match rules
func (b *_Base) checkHandlerFunc(typ reflect.Type, isObj bool) (int, bool) { // 判断是否匹配规则,返回参数个数
	offset := 0
	if isObj {
		offset = 1
	}
	num := typ.NumIn() - offset
	if num == 1 || num == 2 { // Parameter checking 参数检查
		ctxType := typ.In(0 + offset)

		// go-gin default method
		if ctxType == reflect.TypeOf(&gin.Context{}) {
			return num, true
		}

		// Customized context . 自定义的context
		if ctxType == b.apiType {
			return num, true
		}

		// maybe interface
		if b.apiType.ConvertibleTo(ctxType) {
			return num, true
		}

	}
	return num, false
}

// checkHandlerFuncLoose 宽松匹配:只校验参数个数与"第一个参数像上下文"(指针/接口)。
// 用途:显式标注(@Router)或 autoRegister/Exporter 的方法,不该仅仅因为
// 上下文类型没推断准就被静默跳过;类型是否真的匹配交给调用期 ctxArg 判断。
func (b *_Base) checkHandlerFuncLoose(typ reflect.Type, isObj bool) (int, bool) {
	offset := 0
	if isObj {
		offset = 1
	}
	num := typ.NumIn() - offset
	if num != 1 && num != 2 {
		return num, false
	}
	if k := typ.In(offset).Kind(); k == reflect.Ptr || k == reflect.Interface {
		return num, true
	}
	return num, false
}

// ctxArg 按 handler 声明的上下文类型生成入参。
// gin.Context 直接透传;其余走 WithCtx 回调,并校验返回值能否赋给形参类型。
// 类型确实对不上时返回错误,而不是 panic 或静默跳过。
func (b *_Base) ctxArg(want reflect.Type, c *gin.Context) (reflect.Value, error) {
	if want == reflect.TypeOf(&gin.Context{}) {
		return reflect.ValueOf(c), nil
	}
	v := b.apiFun(c)
	rv := reflect.ValueOf(v)
	// 只接受可赋值(含接口实现),不做 Convert:结构相同的两个 context 类型
	// 互相 ConvertibleTo,静默转换会把错误的上下文串进来。
	if rv.IsValid() && rv.Type().AssignableTo(want) {
		return rv, nil
	}
	got := "<nil>"
	if rv.IsValid() {
		got = rv.Type().String()
	}
	return reflect.Value{}, errors.Errorf("ginrpc: ctx type mismatch: method need %v, WithCtx returns %v; use ginrpc.WithCtxType to specify", want.String(), got)
}

// ctxArgOrFail ctxArg 的"出错即结束"封装:详情只进服务端日志,
// 客户端拿到通用错误(不回显内部类型名)。
func (b *_Base) ctxArgOrFail(c *gin.Context, want reflect.Type) (reflect.Value, bool) {
	arg, err := b.ctxArg(want, c)
	if err == nil {
		return arg, true
	}
	mylog.ErrorString(err.Error())
	b.writeResult(c, nil, errors.New("internal server error"))
	return reflect.Value{}, false
}

// HandlerFunc Get and filter the parameters to be bound (object call type)
func (b *_Base) handlerFuncObj(tvl, obj reflect.Value, methodName string) gin.HandlerFunc { // 获取并过滤要绑定的参数(obj 对象类型)
	typ := tvl.Type()
	if typ.NumIn() == 2 { // Parameter checking 参数检查
		ctxType := typ.In(1)

		// 上下文入参统一在调用期适配:gin.Context 与自定义 context 都走这里,
		// 不再在注册期 panic,也不再因为类型没对上被静默跳过。
		return func(c *gin.Context) {
			defer func() {
				if err := recover(); err != nil {
					b.recoverHandler(c, err)
				}
			}()
			arg, ok := b.ctxArgOrFail(c, ctxType)
			if !ok {
				return
			}
			tvl.Call([]reflect.Value{obj, arg})
		}
	}

	// Custom context type with request parameters .自定义的context类型,带request 请求参数
	call, err := b.getCallObj3(tvl, obj, methodName)
	if err != nil { // Direct reporting error.
		panic(err)
	}

	return call
}

// beforCall 仅在真正存在中间件(对象自身实现 GinBeforeAfter 或全局 beforeAfter)
// 时才会被调用;funcName 由注册期算好并闭包捕获,避免每请求 fmt.Sprintf + 反射。
func (b *_Base) beforCall(c *gin.Context, obj reflect.Value, req interface{}, funcName string) (*GinBeforeAfterInfo, bool) {
	// 继承请求上下文:客户端取消/超时能一路传到业务 handler 与中间件,
	// 历史实现用 context.Background() 丢掉了取消信号。
	reqCtx := context.Background()
	if c.Request != nil {
		reqCtx = c.Request.Context()
	}
	info := &GinBeforeAfterInfo{
		C:        c,
		FuncName: funcName, // 注册期算好
		Req:      req,      // 调用前的请求参数
		Context:  reqCtx,   // 占位参数，可用于存储其他参数，前后连接可用
	}

	is := true
	if bfobj, ok := obj.Interface().(GinBeforeAfter); ok { // 本类型
		is = bfobj.GinBefore(info)
	}
	if is && len(b.beforeAfter) > 0 {
		for _, call := range b.beforeAfter {
			is = call.GinBefore(info)
			if !is {
				break
			}
		}
	}
	return info, is
}

func (b *_Base) afterCall(info *GinBeforeAfterInfo, obj reflect.Value) bool {
	is := true
	if bfobj, ok := obj.Interface().(GinBeforeAfter); ok { // 本类型
		is = bfobj.GinAfter(info)
	}
	if is && len(b.beforeAfter) > 0 {
		for _, call := range b.beforeAfter {
			is = call.GinAfter(info)
			if !is {
				break
			}
		}
	}
	return is
}

// Custom context type with request parameters
func (b *_Base) getCallFunc3(tvl reflect.Value) (func(*gin.Context), error) {
	typ := tvl.Type()
	if typ.NumIn() != 2 { // Parameter checking 参数检查
		return nil, errors.New("method " + runtime.FuncForPC(tvl.Pointer()).Name() + " not support!")
	}

	if typ.NumOut() != 0 {
		if typ.NumOut() == 2 { // Parameter checking 参数检查
			if returnType := typ.Out(1); returnType != typeOfError {
				return nil, errors.Errorf("method : %v , returns[1] %v not error",
					runtime.FuncForPC(tvl.Pointer()).Name(), returnType.String())
			}
		} else {
			return nil, errors.Errorf("method : %v , Only 2 return values (obj, error) are supported", runtime.FuncForPC(tvl.Pointer()).Name())
		}
	}

	ctxType, reqType := typ.In(0), typ.In(1)

	reqIsValue := true
	if reqType.Kind() == reflect.Ptr {
		reqIsValue = false
	}

	// hasReturn 注册期确定(NumOut 只可能是 0 或 2,上面已校验)。
	// 无返回值的 handler 自行通过 ctx 写响应,框架不额外输出。
	hasReturn := typ.NumOut() == 2

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				b.recoverHandler(c, err)
			}
		}()

		req := reflect.New(reqType)
		if !reqIsValue {
			req = reflect.New(reqType.Elem())
		}
		if err := b.unmarshal(c, req.Interface()); err != nil { // Return error message.返回错误信息
			b.handErrorString(c, err)
			return
		}

		if reqIsValue {
			req = req.Elem()
		}
		arg, ok := b.ctxArgOrFail(c, ctxType)
		if !ok {
			return
		}
		returnValues := tvl.Call([]reflect.Value{arg, req})
		if !hasReturn { // 无返回值:handler 已自行写响应
			return
		}
		var resp interface{}
		var rerr error
		if len(returnValues) > 0 {
			resp = returnValues[0].Interface()
		}
		if len(returnValues) > 1 && !returnValues[1].IsNil() {
			rerr, _ = returnValues[1].Interface().(error)
		}
		b.writeResult(c, resp, rerr)
	}, nil
}

// Custom context type with request parameters
func (b *_Base) getCallObj3(tvl, obj reflect.Value, methodName string) (func(*gin.Context), error) {
	typ := tvl.Type()
	if typ.NumIn() != 3 { // Parameter checking 参数检查
		return nil, errors.New("method " + runtime.FuncForPC(tvl.Pointer()).Name() + " not support!")
	}

	if typ.NumOut() != 0 {
		if typ.NumOut() == 2 { // Parameter checking 参数检查
			if returnType := typ.Out(1); returnType != typeOfError {
				return nil, errors.Errorf("method : %v , returns[1] %v not error",
					runtime.FuncForPC(tvl.Pointer()).Name(), returnType.String())
			}
		} else {
			return nil, errors.Errorf("method : %v , Only 2 return values (obj, error) are supported", runtime.FuncForPC(tvl.Pointer()).Name())
		}
	}

	ctxType, reqType := typ.In(1), typ.In(2)

	reqIsValue := true
	if reqType.Kind() == reflect.Ptr {
		reqIsValue = false
	}

	// 注册期计算并闭包捕获,避免每请求 fmt.Sprintf / 反射 / 无谓分配:
	// - hasBA:完全没有 before/after 中间件时直接跳过 GinBeforeAfterInfo
	// - funcName:FuncName 注册期算好
	// - hasReturn:NumOut 只可能是 0 或 2
	_, objHasBA := obj.Interface().(GinBeforeAfter)
	hasBA := objHasBA || len(b.beforeAfter) > 0
	funcName := fmt.Sprintf("%v.%v", reflect.Indirect(obj).Type().Name(), methodName)
	hasReturn := typ.NumOut() == 2

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				b.recoverHandler(c, err)
			}
		}()

		req := reflect.New(reqType)
		if !reqIsValue {
			req = reflect.New(reqType.Elem())
		}
		if err := b.unmarshal(c, req.Interface()); err != nil { // Return error message.返回错误信息
			b.handErrorString(c, err)
			return
		}

		if reqIsValue {
			req = req.Elem()
		}

		var bainfo *GinBeforeAfterInfo
		if hasBA {
			var is bool
			bainfo, is = b.beforCall(c, obj, req.Interface(), funcName)
			if !is { // GinBefore 显式拒绝
				c.JSON(http.StatusBadRequest, bainfo.Resp)
				return
			}
		}

		arg, ok := b.ctxArgOrFail(c, ctxType)
		if !ok {
			return
		}
		returnValues := tvl.Call([]reflect.Value{obj, arg, req})
		if !hasReturn { // 无返回值:handler 自行写响应;中间件仅作观测
			if hasBA {
				bainfo.Resp, bainfo.Error = nil, nil
				b.afterCall(bainfo, obj)
			}
			return
		}

		var resp interface{}
		var rerr error
		if len(returnValues) > 0 {
			resp = returnValues[0].Interface()
		}
		if len(returnValues) > 1 && !returnValues[1].IsNil() {
			rerr, _ = returnValues[1].Interface().(error)
		}

		if hasBA {
			// GinAfter 只做观测/改写 resp,不决定"是否输出错误";
			// 采用中间件改写后的结果,错误仍由 writeResult 统一处理。
			bainfo.Resp = resp
			bainfo.Error = rerr
			b.afterCall(bainfo, obj)
			resp = bainfo.Resp
		}

		b.writeResult(c, resp, rerr)
	}, nil
}

// handErrorString 把绑定/校验错误写给客户端。
// 只输出字段名 / Tag() / Param(),绝不输出 err.Value():
// 失败字段可能是 password / access_token / 原始请求内容等敏感值。
func (b *_Base) handErrorString(c *gin.Context, err error) {
	var fields []string
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			tmp := fmt.Sprintf("%v:%v", fe.Field(), fe.Tag())
			if len(fe.Param()) > 0 {
				tmp += fmt.Sprintf("[%v]", fe.Param())
			}
			fields = append(fields, tmp)
		}
	} else if ute, ok := err.(*json.UnmarshalTypeError); ok {
		// 同样不带 ute.Value(可能含原始请求值)
		fields = append(fields, fmt.Sprintf("%v:%v", ute.Field, ute.Type.String()))
	} else {
		fields = append(fields, err.Error())
	}

	msg := message.GetErrorMsg(message.ParameterInvalid)
	msg.Error = fmt.Sprintf("req param : %v", strings.Join(fields, ";"))
	c.JSON(http.StatusBadRequest, msg)
}

// unmarshal 绑定并校验请求参数。
// 绑定错误必须原样返回,不能被 mapJson 的结果覆盖成 nil:
// 历史上 POST+JSON 时 Form 为空,非法请求会以零值对象继续执行并返回 200。
func (b *_Base) unmarshal(c *gin.Context, v interface{}) error {
	isGet := strings.EqualFold(c.Request.Method, http.MethodGet)

	err := c.ShouldBind(v)
	if err != nil && !isGet {
		return err // 非 GET:绑定/校验错误原样返回
	}

	if isGet {
		// GET 的 ShouldBind 走 form binding,只认 form tag / 字段名,
		// 认不了 json tag 的 query 参数,常会在"字段还没被补刀填充"时
		// 误报校验失败,所以这里不采信它的 err。补刀可能改动校验字段,
		// 因此补刀后必须用 gin 现有的 validator 实例重新校验一次
		// (历史上补刀在 validator 之后改字段却不重跑,是另一个漏洞)。
		if merr := mapJson(v, c.Request.Form); merr != nil {
			return merr
		}
		return binding.Validator.ValidateStruct(v)
	}
	return nil
}

// writeResult 统一写回响应:框架始终负责处理 error,成功才 200。
// GinAfter 等中间件只负责观测/改写 resp,不参与"是否输出错误"的决策,
// 函数式与对象式两条 handler 路径都必须经过这里,保证协议一致。
func (b *_Base) writeResult(c *gin.Context, resp interface{}, err error) {
	if c == nil || c.Writer.Written() { // handler 已自行写过响应则不重复写
		return
	}
	if err != nil {
		msg := message.GetErrorMsg(message.InValidOp)
		if st, ok := status.FromError(err); ok { // 只有 grpc status error 才带 code/message
			msg.Code = int(st.Code())
			msg.Error = st.Message()
		} else {
			msg.Error = err.Error()
		}
		c.JSON(grpcCodeToHTTP(err), msg)
		return
	}
	c.JSON(http.StatusOK, resp)
}

// grpcCodeToHTTP 把 grpc 状态码映射为 HTTP 状态码,其余一律按 400 处理。
func grpcCodeToHTTP(err error) int {
	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			return http.StatusBadRequest
		case codes.NotFound:
			return http.StatusNotFound
		case codes.Unauthenticated:
			return http.StatusUnauthorized
		case codes.PermissionDenied:
			return http.StatusForbidden
		case codes.Internal:
			return http.StatusInternalServerError
		}
	}
	return http.StatusBadRequest
}

func (b *_Base) parserStruct(req, resp *parmInfo, astPkg *ast.Package, modPkg, modFile string) (r, p *mydoc.StructInfo) {
	ant := myast.NewStructAnalys(modPkg, modFile)
	if req != nil {
		tmp := astPkg
		if len(req.Pkg) > 0 {
			objFile := myast.EvalSymlinks(modPkg, modFile, req.Import)
			tmp, _ = myast.GetAstPkgs(req.Pkg, objFile) // get ast trees.
		}
		r = ant.ParserStruct(tmp, req.Type)
	}

	if resp != nil {
		tmp := astPkg
		if len(resp.Pkg) > 0 {
			objFile := myast.EvalSymlinks(modPkg, modFile, resp.Import)
			tmp, _ = myast.GetAstPkgs(resp.Pkg, objFile) // get ast trees.
		}
		p = ant.ParserStruct(tmp, resp.Type)
	}

	return
}

var routeRegex = regexp.MustCompile(`@Router\s+(\S+)(?:\s+\[(\S+)\])?`)
var routeRegexParty = regexp.MustCompile(`\[(\S+ \S+)\]`)

func analysisParm(f *ast.FieldList, imports map[string]string, objPkg string, n int) (parm *parmInfo) {
	if f != nil {
		if f.NumFields() > 1 {
			parm = &parmInfo{}
			d := f.List[n].Type
			switch exp := d.(type) {
			case *ast.SelectorExpr: // 非本文件包
				parm.Type = exp.Sel.Name
				if x, ok := exp.X.(*ast.Ident); ok {
					parm.Import = imports[x.Name]
					parm.Pkg = myast.GetImportPkg(parm.Import)
				}
			case *ast.StarExpr: // 本文件
				switch expx := exp.X.(type) {
				case *ast.SelectorExpr: // 非本地包
					parm.Type = expx.Sel.Name
					if x, ok := expx.X.(*ast.Ident); ok {
						parm.Pkg = x.Name
						parm.Import = imports[parm.Pkg]
					}
				case *ast.Ident: // 本文件
					parm.Type = expx.Name
					parm.Import = objPkg // 本包
				default:
					mylog.ErrorString(fmt.Sprintf("not find any expx.(%v) [%v]", reflect.TypeOf(expx), objPkg))
				}
			case *ast.Ident: // 本文件
				parm.Type = exp.Name
				parm.Import = objPkg // 本包
			default:
				mylog.ErrorString(fmt.Sprintf("not find any exp.(%v) [%v]", reflect.TypeOf(d), objPkg))
			}
		}
	}

	if parm != nil {
		if len(parm.Pkg) > 0 {
			var pkg string
			n := strings.LastIndex(parm.Import, "/")
			if n > 0 {
				pkg = parm.Import[n+1:]
			}
			if len(pkg) > 0 {
				parm.Pkg = pkg
			}
		}
	}
	return
}

func (b *_Base) parserComments(f *ast.FuncDecl, objName, objFunc string, imports map[string]string, objPkg string, num int) ([]genComment, *parmInfo, *parmInfo) {
	var note string
	var gcs []genComment
	req := analysisParm(f.Type.Params, imports, objPkg, 1)
	resp := analysisParm(f.Type.Results, imports, objPkg, 0)
	ignore := false

	if f.Doc != nil {
		for _, c := range f.Doc.List {
			gc := genComment{}
			t := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
			if strings.HasPrefix(t, "@Router") {
				// t := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
				matches := routeRegex.FindStringSubmatch(t)
				if len(matches) == 3 {
					gc.RouterPath = matches[1]
					methods := matches[2]
					if methods != "-" {
						if methods == "" {
							gc.Methods = []string{"get"}
						} else {
							gc.Methods = strings.Split(methods, ",")
						}
						// deal 3td party
						for {
							t = t[len(matches[0]):]
							if len(t) > 2 {
								matches = routeRegexParty.FindStringSubmatch(t)
								if len(matches) == 2 {
									tmp := strings.Split(matches[1], " ")
									if len(tmp) == 2 {
										gc.ThirdPartyList = append(gc.ThirdPartyList, GenThirdParty{
											Name: tmp[0],
											Data: tmp[1],
										})
									}
								}
							} else {
								break
							}
						}
						gcs = append(gcs, gc)
					} else {
						ignore = true
						// 记录一条显式忽略标记,写进路由表(getInfo/mp),
						// 供运行期 register 命中即 continue;即使开启
						// WithAutoRegister(true) 兼容模式也不会被兜底注册。
						gcs = append(gcs, genComment{Ignored: true, RouterPath: "-", Methods: []string{"-"}})
					}
				}
				// else {
				// return nil, errors.New("Router information is missing")
				// }
			} else if strings.HasPrefix(t, objFunc) { // find note
				t = strings.TrimSpace(strings.TrimPrefix(t, objFunc))
				note += t
			}
		}

	}

	//defalt
	if len(gcs) == 0 && !ignore {
		gc := genComment{}
		gc.RouterPath, gc.Methods = b.getDefaultComments(objName, objFunc, num)
		gcs = append(gcs, gc)
	}

	// add note 添加注释
	for i := 0; i < len(gcs); i++ {
		gcs[i].Note = note
	}

	return gcs, req, resp
}

// tryGenRegister gen out the Registered config info  by struct object,[prepath + bojname.]
func (b *_Base) tryGenRegister(router gin.IRoutes, cList ...interface{}) bool {
	modPkg, modFile, isFind := myast.GetModuleInfo(2)
	if !isFind {
		return false
	}

	// WithImportFile 的映射是实例级收集的,只在真正做 AST 生成时同步到
	// myast 的全局表(其 EvalSymlinks 直接读全局 map,只支持全局写入);
	// 用 bool 做幂等保护,避免每个 Register 重复写。
	if !b.importFilesApplied {
		for k, v := range b.importFiles {
			myast.AddImportFile(k, v)
		}
		b.importFilesApplied = true
	}

	groupPath := b.BasePath(router)
	doc := mydoc.NewDoc(groupPath)

	for _, c := range cList {
		refVal := reflect.ValueOf(c)
		t := reflect.Indirect(refVal).Type()
		objPkg := t.PkgPath()
		objName := t.Name()

		// now := time.Now()
		// fmt.Println(objPkg, objName)

		// find path
		objFile := myast.EvalSymlinks(modPkg, modFile, objPkg)
		// fmt.Println(objFile)

		astPkgs, _b := myast.GetAstPkgs(objPkg, objFile) // get ast trees.
		if _b {
			// imports := myast.AnalysisImport(astPkgs)
			funMp := myast.GetObjFunMp(astPkgs, objName)
			// ast.Print(token.NewFileSet(), astPkgs)
			// fmt.Println(b)

			refTyp := reflect.TypeOf(c)
			// Install the methods
			for m := 0; m < refTyp.NumMethod(); m++ {
				method := refTyp.Method(m)

				num, _b := b.checkHandlerFunc(method.Type /*.Interface()*/, true)
				loose := false
				if !_b { // 上下文类型没推断准:不再静默丢弃,按宽松规则继续出文档/路由表
					num, _b = b.checkHandlerFuncLoose(method.Type, true)
					loose = _b
				}
				if _b {
					fp, ok := funMp[method.Name]
					if !ok { // AST 里找不到方法体:文档/路由表会缺条目,必须留痕
						mylog.ErrorString(fmt.Sprintf("ginrpc: %v.%v 未在源码中找到,跳过文档与路由表生成", objName, method.Name))
					}
					if ok {
						gcs, req, resp := b.parserComments(fp.AstFunc, objName, method.Name, fp.Imports, objPkg, num)
						if loose && len(gcs) > 0 { // 真正会进文档/路由表时才告警
							mylog.ErrorString(fmt.Sprintf("ginrpc: %v.%v 的上下文类型 %v 与 apiType %v 不匹配,已按宽松模式生成;可用 ginrpc.WithCtxType 显式指定",
								objName, method.Name, method.Type.In(1), b.apiType))
						}
						if b.isOutDoc { // output doc
							// now := time.Now()
							// fmt.Println(objName + "." + method.Name)
							docReq, docResp := b.parserStruct(req, resp, astPkgs, modPkg, modFile)
							// fmt.Println("----------", time.Since(now))
							for _, gc := range gcs {
								if gc.Ignored {
									continue // 忽略的方法不进接口文档
								}
								doc.AddOne(objName, method.Name, gc.RouterPath, gc.Methods, gc.Note, docReq, docResp)
							}
						}

						for _, gc := range gcs {
							if gc.Ignored {
								AddGenIgnore(objName + "." + method.Name)
								continue
							}
							checkOnceAdd(objName+"."+method.Name, gc.RouterPath, gc.Methods, gc.ThirdPartyList, gc.Note)
						}
					}

				}
			}
		}
	}

	if b.isOutDoc {
		doc.GenSwagger(modFile + "/docs/swagger/")
		doc.GenMarkdown(modFile + "/docs/markdown/")
	}
	genOutPut(b.outPath, modFile) // generate code
	return true
}

func (b *_Base) BasePath(router gin.IRoutes) string {
	switch r := router.(type) {
	case *gin.RouterGroup:
		return r.BasePath()
	case *gin.Engine:
		return r.BasePath()
	}
	return ""
}

// cleanRouterPath 校验并规范化路由路径。
// ok=false 表示拒绝注册:空路径/忽略占位、".." 穿越、内部空段("//")、
// 重复的 ":" 通配参数名(gin 对重复的同一段通配符会直接 panic)。
func cleanRouterPath(p string) (string, bool) {
	if p == "" || p == "-" {
		return "", false
	}
	segs := strings.Split(p, "/")
	seen := make(map[string]struct{})
	for i, seg := range segs {
		if seg == ".." || seg == "." {
			return "", false
		}
		if seg == "" && i > 0 && i < len(segs)-1 {
			return "", false // 内部空段 //
		}
		if strings.HasPrefix(seg, ":") {
			if _, ok := seen[seg]; ok {
				return "", false
			}
			seen[seg] = struct{}{}
		}
	}
	return path.Clean(p), true
}

// register Registered by struct object,[prepath + bojname.]
// 默认只注册"显式标注"的方法(路由表里有条目,或对象实现 Exporter),
// 不再对表里没有条目的方法无条件兜底注册,避免内部方法被意外暴露;
// WithAutoRegister(true) 可恢复旧行为。
func (b *_Base) register(router gin.IRoutes, cList ...interface{}) bool {
	mp := getInfo()
	// seen 记录本次注册过的 (method,path),避免路由表里存在重复项时
	// 向 gin 重复注册同一路由直接 panic(handlers are already registered)
	seen := make(map[string]struct{})

	for _, c := range cList {
		refTyp := reflect.TypeOf(c)
		refVal := reflect.ValueOf(c)
		t := reflect.Indirect(refVal).Type()
		objName := t.Name()

		// 对象级 opt-in:实现 Exporter 时,无表项的方法也允许兜底注册
		_, exporter := c.(Exporter)

		// Install the methods
		for m := 0; m < refTyp.NumMethod(); m++ {
			method := refTyp.Method(m)
			num, _b := b.checkHandlerFunc(method.Type /*.Interface()*/, true)

			key := objName + "." + method.Name
			routes, inTable := mp[key]

			// 显式忽略(@Router - [-]):无论 autoRegister 开不开都不注册
			if inTable {
				ignored := false
				for i := range routes {
					if routes[i].GenComment.Ignored {
						ignored = true
						break
					}
				}
				if ignored {
					continue
				}
			}

			// 上下文类型没推断准不再静默跳过:显式标注(@Router)或
			// autoRegister/Exporter 的方法按宽松规则继续注册,
			// 调用期由 ctxArg 适配并给出明确错误。
			if !_b {
				if !inTable && !b.autoRegister && !exporter {
					continue
				}
				var ok bool
				num, ok = b.checkHandlerFuncLoose(method.Type, true)
				if !ok {
					continue
				}
				mylog.ErrorString(fmt.Sprintf("ginrpc: register %v 的上下文类型 %v 与 apiType %v 不匹配,按宽松模式注册;可用 ginrpc.WithCtxType 显式指定",
					key, method.Type.In(1), b.apiType))
			}

			if inTable {
				for i := range routes {
					routerPath, ok := cleanRouterPath(routes[i].GenComment.RouterPath)
					if !ok {
						mylog.ErrorString(fmt.Sprintf("register: skip invalid router path %q for %s", routes[i].GenComment.RouterPath, key))
						continue
					}
					methods := filterRegistered(seen, routes[i].GenComment.Methods, routerPath)
					if len(methods) == 0 {
						continue
					}
					if err := b.registerHandlerObj(router, methods, routerPath, method.Name, method.Func, refVal); err != nil {
						mylog.Error(err) // 路由注册失败不能静默丢弃
					}
				}
				continue
			}

			// 表里无条目:去掉"找不到就默认注册"的兜底,
			// 仅 autoRegister 兼容模式或对象实现 Exporter 时才允许。
			if !b.autoRegister && !exporter {
				continue
			}
			routerPath, methods := b.getDefaultComments(objName, method.Name, num)
			cleaned, ok := cleanRouterPath(routerPath)
			if !ok {
				continue
			}
			methods = filterRegistered(seen, methods, cleaned)
			if len(methods) == 0 {
				continue
			}
			if err := b.registerHandlerObj(router, methods, cleaned, method.Name, method.Func, refVal); err != nil {
				mylog.Error(err)
			}
		}
	}
	return true
}

// filterRegistered 过滤掉本次 Register 中已经注册过的 (method,path)
func filterRegistered(seen map[string]struct{}, methods []string, routerPath string) []string {
	out := make([]string, 0, len(methods))
	for _, m := range methods {
		key := strings.ToUpper(strings.TrimSpace(m)) + " " + routerPath
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, m)
	}
	return out
}

func (b *_Base) getDefaultComments(objName, objFunc string, num int) (routerPath string, methods []string) {
	// 默认方法收窄:一个参数(ctx)→ GET,两个参数(ctx,req)→ POST,
	// 与 README_cn.md 一致;需要多方法必须显式 @Router 注解,不再默认 ANY。
	methods = []string{"get"}
	if num == 2 { // parm 2 , post default
		methods = []string{"post"}
	}

	if b.isBigCamel { // big camel style.大驼峰
		routerPath = objName + "." + objFunc
	} else {
		routerPath = mybigcamel.UnMarshal(objName) + "." + mybigcamel.UnMarshal(objFunc)
	}

	return
}

// registerHandlerObj Multiple registration methods.获取并过滤要绑定的参数
func (b *_Base) registerHandlerObj(router gin.IRoutes, httpMethod []string, relativePath, methodName string, tvl, obj reflect.Value) error {
	call := b.handlerFuncObj(tvl, obj, methodName)

	for _, v := range httpMethod {
		if err := handleHTTPMethod(router, v, relativePath, call); err != nil {
			return err
		}
	}

	return nil
}
