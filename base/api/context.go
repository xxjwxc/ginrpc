package api

/*
* 基础类目 下一个版本将支持 单个struct 自动解析。
 */

import (
	"github.com/gin-gonic/gin"
)

//Context .
type Context struct {
	*gin.Context
}

//Newctx .
func Newctx(c *gin.Context) *Context {
	return &Context{c}
}

//GetVersion 获取版本号
func (c *Context) GetVersion() string {
	return c.Param("version")
}

//获取用户信息
