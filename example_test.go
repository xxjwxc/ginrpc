package main

import (
	"net/http/httptest"
	"testing"

	"github.com/xxjwxc/ginrpc/base/api"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
)

func TestMain(t *testing.T) {

	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	TestFun1(c)
	TestFun2(api.Newctx(c))
	TestFun3(api.Newctx(c), &ReqTest{})
	TestFun4(c, ReqTest{})

	router := gin.Default()
	router.POST("/test1", base.GetHandlerFunc(TestFun1))
	router.POST("/test2", base.GetHandlerFunc(TestFun2))
	router.POST("/test3", base.GetHandlerFunc(TestFun3))
	router.POST("/test4", base.GetHandlerFunc(TestFun4))

	//router.Run(":8080")
}
