package main

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/xxjwxc/ginrpc/base"
)

func TestMain(t *testing.T) {
	router := gin.Default()
	router.POST("/test1", base.GetHandlerFunc(TestFun1))
	router.POST("/test2", base.GetHandlerFunc(TestFun2))
	router.POST("/test3", base.GetHandlerFunc(TestFun3))
	router.POST("/test4", base.GetHandlerFunc(TestFun4))

	//router.Run(":8080")
}
