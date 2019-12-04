package ginrpc

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/xxjwxc/public/serializing"
	"github.com/xxjwxc/public/tools"
)

var _mu sync.Mutex // protects the serviceMap
var _once sync.Once
var _genInfo genInfo

// AddGenOne add one to base case
func AddGenOne(handFunName, routerPath string, methods []string) {
	_mu.Lock()
	defer _mu.Unlock()
	_genInfo.List = append(_genInfo.List, genRouterInfo{
		HandFunName: handFunName,
		genComment: genComment{
			RouterPath: routerPath,
			Methods:    methods,
		},
	})
}

// SetVersion user timestamp to replace version
func SetVersion(tm int64) {
	_mu.Lock()
	defer _mu.Unlock()
	_genInfo.Tm = tm
}

func checkOnceAdd(handFunName, routerPath string, methods []string) {
	_once.Do(func() {
		_mu.Lock()
		defer _mu.Unlock()
		_genInfo.Tm = time.Now().Unix()
		_genInfo.List = []genRouterInfo{} // reset
	})

	AddGenOne(handFunName, routerPath, methods)
}

func GetStringList(list []string) string {
	return `"` + strings.Join(list, `","`) + `"`
}

func genOutPut(outDir, modFile string) {
	_mu.Lock()
	defer _mu.Unlock()

	if len(outDir) == 0 {
		outDir = modFile + "/routers/"
	}
	pkgName := getPkgName(outDir)

	_genInfo.Tm = time.Now().Unix()

	data := struct {
		genInfo
		PkgName string
	}{
		genInfo: _genInfo,
		PkgName: pkgName,
	}

	tmpl, err := template.New("gen_out").Funcs(template.FuncMap{"GetStringList": GetStringList}).Parse(genTemp)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, data)
	tools.WriteFile(outDir+"gen_router.go", []string{buf.String()}, true)

	// format
	exec.Command("gofmt", "-l", "-w", outDir).Output()

	// gob serialize 序列化
	_data, _ := serializing.Encode(_genInfo)
	flag := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	f, err := os.OpenFile(tools.GetModelPath()+"/gen_router.data", flag, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	f.Write(_data)
}

func getPkgName(dir string) string {
	dir = strings.Replace(dir, "\\", "/", -1)
	dir = strings.TrimRight(dir, "/")

	var pkgName string
	list := strings.Split(dir, "/")
	if len(list) > 0 {
		pkgName = list[len(list)-1]
	}

	if len(pkgName) == 0 || pkgName == "." {
		list = strings.Split(tools.GetModelPath(), "/")
		if len(list) > 0 {
			pkgName = list[len(list)-1]
		}
	}

	return pkgName
}
