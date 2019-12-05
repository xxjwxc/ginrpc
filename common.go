package ginrpc

import (
	"fmt"
	"go/ast"
	"net/http"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/public/errors"
	"github.com/xxjwxc/public/mybigcamel"
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
	}
	return num, false
}

// HandlerFunc Get and filter the parameters to be bound (object call type)
func (b *_Base) handlerFuncObj(tvl, obj reflect.Value) gin.HandlerFunc { // 获取并过滤要绑定的参数(obj 对象类型)
	typ := tvl.Type()
	if typ.NumIn() == 1 { // Parameter checking 参数检查
		ctxType := typ.In(0)

		// go-gin default method
		apiFun := func(c *gin.Context) interface{} { return c }
		if ctxType == b.apiType { // Customized context . 自定义的context
			apiFun = b.apiFun
		} else if !(ctxType == reflect.TypeOf(&gin.Context{})) {
			panic("method " + runtime.FuncForPC(tvl.Pointer()).Name() + " not support!")
		}

		return func(c *gin.Context) {
			tvl.Call([]reflect.Value{obj, reflect.ValueOf(apiFun(c))})
		}
	}

	// Custom context type with request parameters .自定义的context类型,带request 请求参数
	call, err := b.getCallFunc3(tvl, obj)
	if err != nil { // Direct reporting error.
		panic(err)
	}

	return call
}

// Custom context type with request parameters
func (b *_Base) getCallFunc3(tvls ...reflect.Value) (func(*gin.Context), error) {
	offset := 0
	if len(tvls) > 1 {
		offset = 1
	}

	tvl := tvls[0]
	typ := tvl.Type()
	if typ.NumIn() != (2 + offset) { // Parameter checking 参数检查
		return nil, errors.New("method " + runtime.FuncForPC(tvl.Pointer()).Name() + " not support!")
	}

	ctxType, reqType := typ.In(0+offset), typ.In(1+offset)

	reqIsGinCtx := false
	if ctxType == reflect.TypeOf(&gin.Context{}) {
		reqIsGinCtx = true
	}

	// ctxType != reflect.TypeOf(gin.Context{}) &&
	// ctxType != reflect.Indirect(reflect.ValueOf(b.iAPIType)).Type()
	if !reqIsGinCtx && ctxType != b.apiType {
		return nil, errors.New("method " + runtime.FuncForPC(tvl.Pointer()).Name() + " first parm not support!")
	}

	// reqIsValue := true
	// if reqType.Kind() == reflect.Ptr {
	// 	reqIsValue = false
	// }
	apiFun := func(c *gin.Context) interface{} { return c }
	if !reqIsGinCtx {
		apiFun = b.apiFun
	}

	return func(c *gin.Context) {
		req := reflect.New(reqType)
		if err := b.unmarshal(c, req.Interface()); err != nil { // Return error message.返回错误信息
			c.JSON(http.StatusBadRequest, gin.H{"state": false, "code": 1001, "error": err.Error()})
			return
		}

		if offset > 0 {
			tvl.Call([]reflect.Value{tvls[1], reflect.ValueOf(apiFun(c)), req.Elem()})
		} else {
			tvl.Call([]reflect.Value{reflect.ValueOf(apiFun(c)), req.Elem()})
		}
	}, nil
}

func (b *_Base) unmarshal(c *gin.Context, v interface{}) error {
	return c.ShouldBind(v)
}

var routeRegex = regexp.MustCompile(`@router\s+(\S+)(?:\s+\[(\S+)\])?`)

func (b *_Base) parserComments(f *ast.FuncDecl, objName, objFunc string, num int) []genComment {
	var gcs []genComment
	if f.Doc != nil {
		for _, c := range f.Doc.List {
			gc := genComment{}
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			if strings.HasPrefix(t, "@router") {
				t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
				matches := routeRegex.FindStringSubmatch(t)
				if len(matches) == 3 {
					gc.RouterPath = matches[1]
					methods := matches[2]
					if methods == "" {
						gc.Methods = []string{"get"}
					} else {
						gc.Methods = strings.Split(methods, ",")
					}
					gcs = append(gcs, gc)
				} else {
					// return nil, errors.New("Router information is missing")
				}
			}
		}
	}

	//defalt
	if len(gcs) == 0 {
		gc := genComment{}
		gc.RouterPath, gc.Methods = b.getDefaultComments(objName, objFunc, num)
		gcs = append(gcs, gc)
	}

	return gcs
}

// tryGenRegister gen out the Registered config info  by struct object,[prepath + bojname.]
func (b *_Base) tryGenRegister(router *gin.Engine, cList ...interface{}) bool {
	modPkg, modFile, isFind := getModuleInfo()
	if !isFind {
		return false
	}

	for _, c := range cList {
		refVal := reflect.ValueOf(c)
		t := reflect.Indirect(refVal).Type()
		objPkg := t.PkgPath()
		objName := t.Name()
		// fmt.Println(objPkg, objName)

		// find path
		objFile := evalSymlinks(modPkg, modFile, objPkg)
		fmt.Println(objFile)

		astPkgs, _b := getAstPkgs(objPkg, objFile) // get ast trees.
		if _b {
			refTyp := reflect.TypeOf(c)
			funMp := make(map[string]*ast.FuncDecl, refTyp.NumMethod())

			// find all exported func of sturct objName
			for _, fl := range astPkgs.Files {
				for _, d := range fl.Decls {
					switch specDecl := d.(type) {
					case *ast.FuncDecl:
						if specDecl.Recv != nil {
							if exp, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr); ok { // Check that the type is correct first beforing throwing to parser
								if strings.Compare(fmt.Sprint(exp.X), objName) == 0 { // is the same struct
									funMp[specDecl.Name.String()] = specDecl // catch
								}
							}
						}
					}
				}
			}

			// end
			// ast.Print(token.NewFileSet(), astPkgs)
			// fmt.Println(b)

			// Install the methods
			for m := 0; m < refTyp.NumMethod(); m++ {
				method := refTyp.Method(m)
				num, _b := b.checkHandlerFunc(method.Type /*.Interface()*/, true)
				if _b {
					if sdl, ok := funMp[method.Name]; ok {
						gcs := b.parserComments(sdl, objName, method.Name, num)
						for _, gc := range gcs {
							checkOnceAdd(objName+"."+method.Name, gc.RouterPath, gc.Methods)
						}
					}
				}
			}
		}
	}

	genOutPut(b.outPath, modFile) // generate code
	return true
}

// register Registered by struct object,[prepath + bojname.]
func (b *_Base) register(router *gin.Engine, cList ...interface{}) bool {
	mp := getInfo()
	for _, c := range cList {
		refTyp := reflect.TypeOf(c)
		refVal := reflect.ValueOf(c)
		t := reflect.Indirect(refVal).Type()
		objName := t.Name()

		// Install the methods
		for m := 0; m < refTyp.NumMethod(); m++ {
			method := refTyp.Method(m)
			num, _b := b.checkHandlerFunc(method.Type /*.Interface()*/, true)
			if _b {
				if v, ok := mp[objName+"."+method.Name]; ok {
					for _, v1 := range v {
						b.registerHandlerObj(router, v1.Methods, buildRelativePath(b.groupPath, v1.RouterPath), method.Func, refVal)
					}
				} else { // not find using defualt case
					routerPath, methods := b.getDefaultComments(objName, method.Name, num)
					b.registerHandlerObj(router, methods, buildRelativePath(b.groupPath, routerPath), method.Func, refVal)
				}
			}
		}
	}
	return true
}

func (b *_Base) getDefaultComments(objName, objFunc string, num int) (routerPath string, methods []string) {
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
func (b *_Base) registerHandlerObj(router *gin.Engine, httpMethod []string, relativePath string, tvl, obj reflect.Value) error {
	call := b.handlerFuncObj(tvl, obj)

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
			router.POST(relativePath, call)
		case "GET":
			router.GET(relativePath, call)
		case "DELETE":
			router.DELETE(relativePath, call)
		case "PATCH":
			router.PATCH(relativePath, call)
		case "PUT":
			router.PUT(relativePath, call)
		case "OPTIONS":
			router.OPTIONS(relativePath, call)
		case "HEAD":
			router.HEAD(relativePath, call)
		case "ANY":
			router.Any(relativePath, call)
		default:
			return errors.Errorf("method:[%v] not support", httpMethod)
		}
	}

	return nil
}
