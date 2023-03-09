package ginrpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/xxjwxc/public/message"
	"github.com/xxjwxc/public/serializing"
	"github.com/xxjwxc/public/tools"
)

const (
	getRouter = "/conf/gen_router.data"
)

var _mu sync.Mutex // protects the serviceMap
var _once sync.Once
var _genInfo genInfo
var _genInfoCnf genInfo
var _genMap map[string]GenThirdParty

func init() {
	_genMap = make(map[string]GenThirdParty)
	data, err := ioutil.ReadFile(path.Join(tools.GetCurrentDirectory(), getRouter))
	if err == nil {
		serializing.Decode(data, &_genInfoCnf) // gob de serialize 反序列化
	}
}

// AddGenOne add one to base case
func AddGenOne(handFunName, routerPath string, methods []string, thirdParty []GenThirdParty, note string) {
	_mu.Lock()
	defer _mu.Unlock()
	_genInfo.List = append(_genInfo.List, genRouterInfo{
		HandFunName: handFunName,
		GenComment: genComment{
			Note:           note,
			RouterPath:     routerPath,
			Methods:        methods,
			ThirdPartyList: thirdParty,
		},
	})
	for _, v := range thirdParty {
		_genMap[fmt.Sprintf("%v-%v", routerPath, v.Name)] = GenThirdParty{
			Note: note,
			Name: v.Name,
			Data: v.Data,
		}
	}
}

func GetThirdParty(routerPath, thirdParty string) (*GenThirdParty, error) {
	if _, ok := _genMap[fmt.Sprintf("%v-%v", routerPath, thirdParty)]; ok {
		tmp := _genMap[routerPath]
		return &GenThirdParty{
			Name: tmp.Name,
			Note: tmp.Note,
			Data: tmp.Data,
		}, nil
	}
	return nil, message.GetError(message.NotFindError)
}
func GetAllRouter() []genRouterInfo {
	_mu.Lock()
	defer _mu.Unlock()
	return _genInfo.List
}

// SetVersion user timestamp to replace version
func SetVersion(tm int64) {
	_mu.Lock()
	defer _mu.Unlock()
	_genInfo.Tm = tm
}

func GetVersion() string {
	return fmt.Sprintf("%v", _genInfo.Tm)
}

func checkOnceAdd(handFunName, routerPath string, methods []string, thirdParty []GenThirdParty, note string) {
	_once.Do(func() {
		_mu.Lock()
		defer _mu.Unlock()
		_genInfo.Tm = time.Now().Unix()
		_genInfo.List = []genRouterInfo{} // reset
	})

	AddGenOne(handFunName, routerPath, methods, thirdParty, note)
}

// GetStringList format string
func GetStringList(list []string) string {
	return `"` + strings.Join(list, `","`) + `"`
}

// GetPartyList format string
func GetPartyList(list []GenThirdParty) string {
	var tmp []string
	for _, v := range list {
		tmp = append(tmp, fmt.Sprintf(`{Name: "%v", Data: "%v"}`, v.Name, v.Data))
	}
	return strings.Join(tmp, ",")
}

func GetNote(note string) string {
	return fmt.Sprintf("`%v`", note)
}

func genOutPut(outDir, modFile string) {
	_mu.Lock()
	defer _mu.Unlock()

	b := genCode(outDir, modFile) // gen .go file

	_genInfo.Tm = time.Now().Unix()
	_data, _ := serializing.Encode(&_genInfo) // gob serialize 序列化
	_path := path.Join(tools.GetCurrentDirectory(), getRouter)
	if !b {
		tools.BuildDir(_path)
	}
	f, err := os.Create(_path)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(_data)
}

func genCode(outDir, modFile string) bool {
	_genInfo.Tm = time.Now().Unix()
	if len(outDir) == 0 {
		outDir = modFile + "/routers/"
	}
	pkgName := getPkgName(outDir)
	data := struct {
		genInfo
		PkgName string
	}{
		genInfo: _genInfo,
		PkgName: pkgName,
	}

	tmpl, err := template.New("gen_out").Funcs(template.FuncMap{"GetStringList": GetStringList, "GetPartyList": GetPartyList, "GetNote": GetNote}).Parse(genTemp)
	if err != nil {
		panic(err)
	}
	var buf bytes.Buffer
	tmpl.Execute(&buf, data)
	f, err := os.Create(outDir + "gen_router.go")
	if err != nil {
		return false
	}
	defer f.Close()
	f.Write(buf.Bytes())

	// format
	exec.Command("gofmt", "-l", "-w", outDir).Output()
	return true
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
		list = strings.Split(tools.GetCurrentDirectory(), "/")
		if len(list) > 0 {
			pkgName = list[len(list)-1]
		}
	}

	return pkgName
}

func getInfo() map[string][]genRouterInfo {
	_mu.Lock()
	defer _mu.Unlock()

	genInfo := _genInfo
	if _genInfoCnf.Tm > genInfo.Tm { // config to update more than coding
		genInfo = _genInfoCnf
	}

	mp := make(map[string][]genRouterInfo, len(genInfo.List))
	for _, v := range genInfo.List {
		tmp := v
		mp[tmp.HandFunName] = append(mp[tmp.HandFunName], tmp)
	}
	return mp
}

func buildRelativePath(prepath, routerPath string) string {
	if strings.HasSuffix(prepath, "/") {
		if strings.HasPrefix(routerPath, "/") {
			return prepath + strings.TrimPrefix(routerPath, "/")
		}
		return prepath + routerPath
	}

	if strings.HasPrefix(routerPath, "/") {
		return prepath + routerPath
	}

	return prepath + "/" + routerPath
}
