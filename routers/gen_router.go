package routers

import (
	"github.com/xxjwxc/ginrpc"
)

func init() {
	ginrpc.SetVersion(1581961792)
	ginrpc.AddGenOne("Hello.HelloS", "/block", []string{"post", "get"})
	ginrpc.AddGenOne("Hello.HelloS2", "Hello.HelloS2", []string{"post"})
	ginrpc.AddGenOne("Hello.HelloS3", "Hello.HelloS3", []string{"post"})
}
