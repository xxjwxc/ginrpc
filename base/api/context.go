// Package api The next version of the underlying category will support automatic parsing of a single struct.
package api

import (
	"github.com/gin-gonic/gin"
)

// Context Wrapping gin context to custom context
type Context struct { // 包装gin的上下文到自定义context
	*gin.Context
}

// NewCtx Create a new custom context
func NewCtx(c *gin.Context) *Context { // 新建一个自定义context
	return &Context{c}
}

// GetVersion Get the version by req url
func (c *Context) GetVersion() string { // 获取版本号
	return c.Param("version")
}
