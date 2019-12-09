package routers

import (
	"github.com/xxjwxc/ginrpc"
)

func init() {
	ginrpc.SetVersion(1575867136)
	ginrpc.AddGenOne("Hello.HelloS", "/block", []string{"post", "get"})
	ginrpc.AddGenOne("Hello.HelloS2", "/block1", []string{"post", "get"})
}
