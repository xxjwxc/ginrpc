package routers

import (
	"github.com/xxjwxc/ginrpc"
)

func init() {
	ginrpc.SetVersion(1575457686)
	ginrpc.AddGenOne("Hello.HelloS", "/block", []string{"post", "get"})
	ginrpc.AddGenOne("Hello.HelloS2", "hello.hello_s2", []string{"post"})
}
