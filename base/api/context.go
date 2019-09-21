// Package api The next version of the underlying category will support automatic parsing of a single struct.
package api

// 基础类目 下一个版本将支持 单个struct 自动解析。

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

// GetVersion Get the version by req ruter
func (c *Context) GetVersion() string { // 获取版本号
	return c.Param("version")
}
