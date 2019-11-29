// Package api The next version of the underlying category will support automatic parsing of a single struct.
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Context Wrapping gin context to custom context
type Context struct { // 包装gin的上下文到自定义context
	*gin.Context
}

// GetVersion Get the version by req url
func (c *Context) GetVersion() string { // 获取版本号
	return c.Param("version")
}

// NewCtx Create a new custom context
func NewCtx(c *gin.Context) *Context { // 新建一个自定义context
	return &Context{c}
}

// NewAPIFunc default of custom handlefunc
func NewAPIFunc(c *gin.Context) interface{} {
	return NewCtx(c)
}

type Hello struct {
}

// @router /block [post]
func (s *Hello) HelloS(c *Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

func (s *Hello) HelloS2(c *Context, req *ReqTest1) {
	fmt.Println(c.Params)
	fmt.Println(req)
	c.JSON(http.StatusOK, "ok")
}

// ReqTest test req
type ReqTest1 struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}
