package ginrpc

import "sync"

var _mu sync.Mutex // protects the serviceMap
var _once sync.Once
var _list []genRouterInfo

// AddGenOne add one to base case
func AddGenOne(handFunName, routerPath string, methods []string) {
	_mu.Lock()
	defer _mu.Unlock()
	_list = append(_list, genRouterInfo{
		handFunName: handFunName,
		genComment: genComment{
			routerPath: routerPath,
			methods:    methods,
		},
	})
}

func checkOnceAdd(handFunName, routerPath string, methods []string) {
	_once.Do(func() {
		_mu.Lock()
		defer _mu.Unlock()
		_list = []genRouterInfo{} // reset
	})

	AddGenOne(handFunName, routerPath, methods)
}
