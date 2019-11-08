package ginrpc

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/api"
)

// ReqTest req test
type ReqTest struct {
	AccessToken string `json:"access_token"`                 // access_token
	UserName    string `json:"user_name" binding:"required"` // user name
	Password    string `json:"password"`                     // password
}

func TestFun(t *testing.T) {
	ginrpc := Default()
	ginrpc.GetHandlerFunc(func(c *gin.Context) {
		fmt.Println(c.Params)
		c.String(200, "ok")
	})

	ginrpc.GetHandlerFunc(func(c *api.Context) {
		fmt.Println(c.Params)
		c.JSON(http.StatusOK, "ok")
	})

	ginrpc.GetHandlerFunc(func(c *api.Context, req *ReqTest) {
		fmt.Println(c.Params)
		fmt.Println(req)
		c.JSON(http.StatusOK, "ok")
	})

	ginrpc.GetHandlerFunc(func(c *gin.Context, req ReqTest) {
		fmt.Println(c.Params)
		fmt.Println(req)

		c.JSON(http.StatusOK, req)
	})
}
