package ginrpc

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// 回归用例:WithCtx 返回的自定义上下文类型没有被 WithCtxType 显式指定时,
// 历史实现会把 checkHandlerFunc 判为不匹配 -> 路由不注册、接口文档不生成,
// 而且全过程没有日志。这里要求自动推断 + 宽松注册 + 调用期明确报错。

type autoRegCtx struct {
	*gin.Context
}

func (c *autoRegCtx) Hello() string { return "hello" }

type autoRegOtherCtx struct {
	*gin.Context
}

type autoRegReq struct {
	Name string `json:"name"`
}

type autoRegResp struct {
	Msg string `json:"msg"`
}

type autoRegDemo struct{}

func (d *autoRegDemo) CtxOnly(c *autoRegCtx) {
	c.String(http.StatusOK, c.Hello())
}

func (d *autoRegDemo) CtxReq(c *autoRegCtx, req *autoRegReq) (*autoRegResp, error) {
	return &autoRegResp{Msg: req.Name}, nil
}

type autoRegBadDemo struct{}

func (d *autoRegBadDemo) CtxOnly(c *autoRegOtherCtx) {
	c.String(http.StatusOK, "should not reach")
}

func findRouteByMethod(r *gin.Engine, method string) string {
	for _, rt := range r.Routes() {
		if rt.Method == method {
			return rt.Path
		}
	}
	return ""
}

// TestCustomCtxNotSkipped 自定义上下文(未显式 WithCtxType)必须能注册并正常调用。
func TestCustomCtxNotSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	b := New(
		WithCtx(func(c *gin.Context) interface{} { return &autoRegCtx{c} }),
		WithAutoRegister(true),
	)
	b.Register(r, &autoRegDemo{})

	if len(r.Routes()) != 2 {
		t.Fatalf("自定义上下文方法被跳过,期望 2 条路由,实际 %d: %v", len(r.Routes()), r.Routes())
	}

	getPath := findRouteByMethod(r, http.MethodGet)
	if getPath == "" {
		t.Fatalf("CtxOnly 未注册: %v", r.Routes())
	}
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, getPath, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != "hello" {
		t.Fatalf("CtxOnly 响应异常: code=%d body=%q", w.Code, w.Body.String())
	}

	postPath := findRouteByMethod(r, http.MethodPost)
	if postPath == "" {
		t.Fatalf("CtxReq 未注册: %v", r.Routes())
	}
	body, _ := json.Marshal(&autoRegReq{Name: "hai"})
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, postPath, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"msg":"hai"`) {
		t.Fatalf("CtxReq 响应异常: code=%d body=%q", w.Code, w.Body.String())
	}
}

// TestCtxMismatchNotSkipped 上下文类型确实对不上时:路由照常注册,
// 调用期返回明确错误(不再静默跳过,也不再注册期 panic)。
func TestCtxMismatchNotSkipped(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	b := New(
		WithCtx(func(c *gin.Context) interface{} { return &autoRegCtx{c} }),
		WithAutoRegister(true),
	)
	b.Register(r, &autoRegBadDemo{})

	if len(r.Routes()) != 1 {
		t.Fatalf("类型不匹配的 handler 被跳过: %v", r.Routes())
	}

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, r.Routes()[0].Path, nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望 400,实际 %d body=%q", w.Code, w.Body.String())
	}
}
