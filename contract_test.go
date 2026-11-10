package ginrpc

// 运行期契约测试:用 httptest + router.ServeHTTP 真正发请求并断言状态码/响应体,
// 补上历史测试"只调 Register、从不 ServeHTTP、无断言"导致的 P0 漏洞漏网。

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// contractReq 带 required 校验字段,用于验证 unmarshal 的绑定/校验链路。
type contractReq struct {
	Name  string `json:"name" binding:"required"`
	Token string `json:"token"`
	Count int    `json:"count"`
}

func contractRouter() (*gin.Engine, *_Base) {
	gin.SetMode(gin.TestMode)
	return gin.New(), New()
}

func postJSON(router *gin.Engine, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

// TestContractPostBinding:POST 正常 / 非法 JSON 400 / validator 失败 400。
// 覆盖 P0-1(unmarshal 绑定错误不再被 mapJson 覆盖)。
func TestContractPostBinding(t *testing.T) {
	router, base := contractRouter()
	router.POST("/echo", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return &req, nil
	}))

	// 正常
	w := postJSON(router, "/echo", `{"name":"bob","token":"t","count":3}`)
	if w.Code != http.StatusOK {
		t.Fatalf("valid POST: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	var got contractReq
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("valid POST: bad json: %v (%s)", err, w.Body.String())
	}
	if got.Name != "bob" || got.Count != 3 {
		t.Fatalf("valid POST: unexpected body %s", w.Body.String())
	}

	// 非法 JSON:历史上这里会以零值对象继续执行并返回 200
	w = postJSON(router, "/echo", `{not-json`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid JSON: got %d want 400, body=%s", w.Code, w.Body.String())
	}

	// validator 失败(缺少 required 的 name)
	w = postJSON(router, "/echo", `{"token":"t"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validator failure: got %d want 400, body=%s", w.Code, w.Body.String())
	}
	// 校验错误不能回显字段值(err.Value())
	if strings.Contains(w.Body.String(), "but[") {
		t.Fatalf("validation error leaked value: %s", w.Body.String())
	}
}

// TestContractGetPatch:GET 的 mapJson 补刀生效,且补刀后重新校验。
// 覆盖 P0-1 的 "GET 在 validator 之后再改字段也不重新校验"。
func TestContractGetPatch(t *testing.T) {
	router, base := contractRouter()
	router.GET("/echo", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return &req, nil
	}))

	// json tag 的 query 参数由补刀填充(form binding 只认 form tag/字段名)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/echo?name=alice&token=tk&count=7", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("valid GET: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	var got contractReq
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("valid GET: bad json: %v (%s)", err, w.Body.String())
	}
	if got.Name != "alice" || got.Count != 7 {
		t.Fatalf("GET patch failed: %s", w.Body.String())
	}

	// 补刀后必须重新校验:缺少 required 字段不能被当成 200
	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/echo?token=tk", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("GET missing required: got %d want 400, body=%s", w.Code, w.Body.String())
	}
}

// TestContractHandlerError:函数式 handler 返回 error 时,框架统一写非 200,
// 并把 grpc code 映射为对应 HTTP 状态。覆盖 P0-2(writeResult)。
func TestContractHandlerError(t *testing.T) {
	router, base := contractRouter()
	router.POST("/bizerr", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, errors.New("biz failed")
	}))
	router.POST("/notfound", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, status.Error(codes.NotFound, "no such user")
	}))
	router.POST("/internal", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, status.Error(codes.Internal, "db down")
	}))
	router.POST("/denied", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, status.Error(codes.PermissionDenied, "denied")
	}))
	router.POST("/unauth", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, status.Error(codes.Unauthenticated, "login please")
	}))
	router.POST("/badarg", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		return nil, status.Error(codes.InvalidArgument, "bad arg")
	}))

	cases := []struct {
		path string
		want int
	}{
		{"/bizerr", http.StatusBadRequest},
		{"/badarg", http.StatusBadRequest},
		{"/notfound", http.StatusNotFound},
		{"/unauth", http.StatusUnauthorized},
		{"/denied", http.StatusForbidden},
		{"/internal", http.StatusInternalServerError},
	}
	for _, tc := range cases {
		w := postJSON(router, tc.path, `{"name":"x"}`)
		if w.Code != tc.want {
			t.Fatalf("%s: got %d want %d, body=%s", tc.path, w.Code, tc.want, w.Body.String())
		}
	}
}

// contractObj 对象式 handler:验证 P0-2 里"对象式返回 error 却 200 + 零值对象"的修复。
type contractObj struct{}

func (o *contractObj) Fail(c *gin.Context, req contractReq) (*contractReq, error) {
	return nil, status.Error(codes.PermissionDenied, "denied")
}

// TestContractObjectError:对象式 handler 返回 error 也必须写非 200(两条路径协议一致)。
func TestContractObjectError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api")
	base := New(WithAutoRegister(true))
	base.Register(group, &contractObj{})

	var target string
	for _, r := range router.Routes() {
		if r.Method == http.MethodPost {
			target = r.Path
			break
		}
	}
	if target == "" {
		t.Fatal("object handler was not registered")
	}
	w := postJSON(router, target, `{"name":"x"}`)
	if w.Code == http.StatusOK {
		t.Fatalf("object handler error must not be 200, got %d body=%s", w.Code, w.Body.String())
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("object handler error: got %d want 403, body=%s", w.Code, w.Body.String())
	}
}

// contractOkObj 对象式成功 handler,用于验证中间件改写后响应结构仍兼容。
type contractOkObj struct{}

func (o *contractOkObj) Ok(c *gin.Context, req contractReq) (*contractReq, error) {
	return &req, nil
}

// TestContractObjectSuccessWithMiddleware:GinAfter 只做观测/改写,
// 成功仍写 200,且响应结构保持 message.MessageBody 兼容(state:true / data)。
func TestContractObjectSuccessWithMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api")
	base := New(WithAutoRegister(true), WithBeforeAfter(&DefaultGinBeforeAfter{}))
	base.Register(group, &contractOkObj{})

	var target string
	for _, r := range router.Routes() {
		if r.Method == http.MethodPost {
			target = r.Path
			break
		}
	}
	if target == "" {
		t.Fatal("object handler was not registered")
	}
	w := postJSON(router, target, `{"name":"x","count":2}`)
	if w.Code != http.StatusOK {
		t.Fatalf("object success: got %d want 200, body=%s", w.Code, w.Body.String())
	}
	var msg struct {
		State bool        `json:"state"`
		Code  int         `json:"code"`
		Data  contractReq `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("object success: bad json: %v (%s)", err, w.Body.String())
	}
	if !msg.State || msg.Data.Name != "x" || msg.Data.Count != 2 {
		t.Fatalf("message.MessageBody compatibility broken: %s", w.Body.String())
	}
}

// TestContractPanicRecover:panic 返回 500 + 固定文案,不泄漏 panic 内容/堆栈。
// 覆盖 P0-4。
func TestContractPanicRecover(t *testing.T) {
	router, base := contractRouter()
	router.POST("/panic", base.HandlerFunc(func(c *gin.Context, req contractReq) (*contractReq, error) {
		panic("SECRET_PANIC_MARKER password=abcd1234")
	}))

	w := postJSON(router, "/panic", `{"name":"x"}`)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("panic: got %d want 500, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "SECRET_PANIC_MARKER") || strings.Contains(body, "password=abcd1234") {
		t.Fatalf("panic content leaked to client: %s", body)
	}
	if strings.Contains(body, "goroutine") || strings.Contains(body, "runtime/debug") || strings.Contains(body, ".go:") {
		t.Fatalf("stack trace leaked to client: %s", body)
	}

	var msg struct {
		State bool   `json:"state"`
		Code  int    `json:"code"`
		Error string `json:"error"`
		Data  string `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("panic: bad json: %v (%s)", err, body)
	}
	if msg.Error != "internal server error" {
		t.Fatalf("panic: want fixed message, got %q", msg.Error)
	}
	if msg.Data == "" {
		t.Fatalf("panic: missing errorID, body=%s", body)
	}
}

// contractIgnoreCtrl 用于验证 @Router - [-] 忽略注解在运行期确实不注册。
type contractIgnoreCtrl struct{}

// @Router /ctr/ok [get]
func (o *contractIgnoreCtrl) Ok(ctx *gin.Context) {
	ctx.String(http.StatusOK, "ok")
}

// @Router - [-]
func (o *contractIgnoreCtrl) Secret(ctx *gin.Context) {
	ctx.String(http.StatusOK, "secret")
}

// TestContractIgnoredRouter:被显式忽略的方法不进路由表(即使开启兼容模式也不会被兜底注册)。
// 覆盖 P0-3。
func TestContractIgnoredRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	base := New(
		WithDebug(true), // 触发 AST 扫描以生成路由表
		WithOutPath(filepath.ToSlash(t.TempDir())+"/"), // 生成产物写到临时目录,不碰工作区
		WithAutoRegister(true),                         // 即便兜底开着,忽略也应生效
	)
	base.Register(router, &contractIgnoreCtrl{})

	routes := router.Routes()
	if len(routes) != 1 {
		t.Fatalf("expected exactly 1 route (Ok only), got %d: %+v", len(routes), routes)
	}
	if routes[0].Method != http.MethodGet || routes[0].Path != "/ctr/ok" {
		t.Fatalf("unexpected route: %s %s", routes[0].Method, routes[0].Path)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ctr/ok", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("Ok route: got %d want 200", w.Code)
	}
}

// contractMethodCtrl 单参数(ctx)方法,用于验证默认方法收窄为 GET。
type contractMethodCtrl struct{}

func (o *contractMethodCtrl) Only(ctx *gin.Context) {
	ctx.String(http.StatusOK, "only")
}

// TestContractDefaultMethodNarrow:num==1 默认只注册 GET,DELETE 不接受。
// 覆盖 P1-5。
func TestContractDefaultMethodNarrow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	group := router.Group("/api")
	base := New(WithAutoRegister(true))
	base.Register(group, &contractMethodCtrl{})

	// 默认方法收窄:所有已注册的方法都应是 GET,不能出现 ANY/DELETE/PUT 等。
	var path string
	for _, r := range router.Routes() {
		if r.Method != http.MethodGet {
			t.Fatalf("1-param default should be GET only, got method %s for %s", r.Method, r.Path)
		}
		if path == "" {
			path = r.Path
		}
	}
	if path == "" {
		t.Fatal("no GET route registered")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("GET: got %d want 200", w.Code)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, path, nil))
	if w.Code == http.StatusOK {
		t.Fatalf("DELETE must not reach the GET-only handler, got %d body=%s", w.Code, w.Body.String())
	}
}
