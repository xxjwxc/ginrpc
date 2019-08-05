package base

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base/api"
)

//ReqTest .
type ReqTest struct {
	AccessToken string `json:"access_token"`                 //access_token
	UserName    string `json:"user_name" binding:"required"` //用户名
	Password    string `json:"password"`                     //新密码
}

func TestFun(t *testing.T) {
	GetHandlerFunc(func(c *gin.Context) {
		fmt.Println(c.Params)
		c.String(200, "ok")
	})

	GetHandlerFunc(func(c *api.Context) {
		fmt.Println(c.Params)
		c.JSON(http.StatusOK, "ok")
	})

	GetHandlerFunc(func(c *api.Context, req *ReqTest) {
		fmt.Println(c.Params)
		fmt.Println(req)
		c.JSON(http.StatusOK, "ok")
	})

	GetHandlerFunc(func(c *gin.Context, req ReqTest) {
		fmt.Println(c.Params)
		fmt.Println(req)

		c.JSON(http.StatusOK, req)
	})
}
