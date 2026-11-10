package ginrpc

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/xxjwxc/public/message"
	"github.com/xxjwxc/public/mylog"
	"github.com/xxjwxc/public/serializing"
	"github.com/xxjwxc/public/tools"
)

const (
	getRouter = "/conf/gen_router.data"
)

// 注意:_genInfo / _genInfoCnf / _genMap / _once / _cnfOnce 必须是包级状态,
// 不能下沉为 _Base 实例字段:生成的 routers/gen_router.go 通过
// ginrpc.AddGenOne / ginrpc.SetVersion / ginrpc.AddGenIgnore 这些**包级函数**
// 在 init() 里回填路由表,跨实例共享是这套生成机制的契约。
// 代价是同一进程内多个 _Base 实例共享同一张表(串台),因此所有写入都在 _mu 下,
// 且 AddGenOne 按 (HandFunName, RouterPath) 去重,避免重复注册直接 panic。
var _mu sync.Mutex // protects _genInfo / _genInfoCnf / _genMap
var _once sync.Once
var _cnfOnce sync.Once
var _genInfo genInfo
var _genInfoCnf genInfo

var _genMap = make(map[string]GenThirdParty)

// loadGenConf 只读懒加载 conf/gen_router.data。
// 该文件是可选的"预生成路由表"输入,不是版本库硬依赖:找不到/解码失败
// 都当作空表返回,不报错(生产只读文件系统、go test 都走这条路径)。
// 生成产物统一写到显式 outPath(见 genOutPut),不再往 cwd 写。
func loadGenConf() {
	_cnfOnce.Do(func() {
		cwd := tools.GetCurrentDirectory()
		if len(cwd) == 0 {
			return
		}

		data, err := os.ReadFile(path.Join(cwd, getRouter))
		if err != nil {
			return // 没有这个文件是正常情况
		}

		var cnf genInfo
		if err := serializing.Decode(data, &cnf); err != nil {
			mylog.Error(err)
			return
		}

		_mu.Lock()
		_genInfoCnf = cnf
		_mu.Unlock()
	})
}

// AddGenOne add one to base case
func AddGenOne(handFunName, routerPath string, methods []string, thirdParty []GenThirdParty, note string) {
	_mu.Lock()
	defer _mu.Unlock()

	// 同一个 handler + 同一个路由路径只保留一份。
	// 历史实现无条件 append:多个 controller 重复 Register(或同一个方法带重复注解)时,
	// 路由表里会出现重复项,随后向 gin 重复注册同一路由直接 panic
	// ("handlers are already registered for path ...")。
	for i := range _genInfo.List {
		if _genInfo.List[i].HandFunName == handFunName && _genInfo.List[i].GenComment.RouterPath == routerPath {
			_genInfo.List[i].GenComment = genComment{
				Note:           note,
				RouterPath:     routerPath,
				Methods:        methods,
				ThirdPartyList: thirdParty,
			}
			addThirdPartyLocked(routerPath, thirdParty, note)
			return
		}
	}

	_genInfo.List = append(_genInfo.List, genRouterInfo{
		HandFunName: handFunName,
		GenComment: genComment{
			Note:           note,
			RouterPath:     routerPath,
			Methods:        methods,
			ThirdPartyList: thirdParty,
		},
	})
	addThirdPartyLocked(routerPath, thirdParty, note)
}

// AddGenIgnore 记录一条被 @Router - [-] 显式忽略的方法。
// 单独登记(而不是靠"表里没有条目"来推断),保证即使开启
// WithAutoRegister(true) 兼容模式,被忽略的方法也不会被兜底注册。
// 供生成的 routers/gen_router.go 的 init() 调用。
func AddGenIgnore(handFunName string) {
	_mu.Lock()
	defer _mu.Unlock()

	for i := range _genInfo.List {
		if _genInfo.List[i].HandFunName == handFunName {
			_genInfo.List[i].GenComment.Ignored = true
			return
		}
	}
	_genInfo.List = append(_genInfo.List, genRouterInfo{
		HandFunName: handFunName,
		GenComment:  genComment{Ignored: true, RouterPath: "-", Methods: []string{"-"}},
	})
}

// addThirdPartyLocked 写第三方注解路由缓存,调用方必须已持有 _mu
func addThirdPartyLocked(routerPath string, thirdParty []GenThirdParty, note string) {
	for _, v := range thirdParty {
		_genMap[fmt.Sprintf("%v-%v", routerPath, v.Name)] = GenThirdParty{
			Note: note,
			Name: v.Name,
			Data: v.Data,
		}
	}
}

func GetThirdParty(routerPath, thirdParty string) (*GenThirdParty, error) {
	_mu.Lock()
	tmp, ok := _genMap[fmt.Sprintf("%v-%v", routerPath, thirdParty)]
	_mu.Unlock()

	if ok {
		return &GenThirdParty{
			Name: tmp.Name,
			Note: tmp.Note,
			Data: tmp.Data,
		}, nil
	}
	return nil, message.GetError(message.NotFindError)
}

// GetAllRouter 返回已注册路由表。
// 返回副本,避免调用方修改到内部切片(历史实现直接返回内部 slice)。
func GetAllRouter() []genRouterInfo {
	_mu.Lock()
	defer _mu.Unlock()

	if len(_genInfo.List) == 0 {
		return nil
	}
	out := make([]genRouterInfo, len(_genInfo.List))
	copy(out, _genInfo.List)
	return out
}

// SetVersion user timestamp to replace version
func SetVersion(tm int64) {
	_mu.Lock()
	defer _mu.Unlock()
	_genInfo.Tm = tm
}

func GetVersion() string {
	_mu.Lock()
	tm := _genInfo.Tm
	_mu.Unlock()
	return fmt.Sprintf("%v", tm)
}

// checkOnceAdd 记录一条注解路由。
// 历史上 _once.Do 里无条件 `_genInfo.List = []genRouterInfo{}`,
// 会把 generated routers/gen_router.go 的 init()(通过 AddGenOne/AddGenIgnore)
// 预先登记的路由一并抹掉,造成静默丢路由。AddGenOne 现在按
// (HandFunName, RouterPath) 去重/更新,这里只需刷新一次 Tm,不再清空列表。
func checkOnceAdd(handFunName, routerPath string, methods []string, thirdParty []GenThirdParty, note string) {
	_once.Do(func() {
		_mu.Lock()
		_genInfo.Tm = time.Now().Unix()
		_mu.Unlock()
	})

	AddGenOne(handFunName, routerPath, methods, thirdParty, note)
}

// GetStringList format string
func GetStringList(list []string) string {
	tmp := make([]string, 0, len(list))
	for _, v := range list {
		tmp = append(tmp, strconv.Quote(v)) // 用 %q 转义,避免注解里的特殊字符破坏生成代码
	}
	return strings.Join(tmp, ",")
}

// GetPartyList format string
func GetPartyList(list []GenThirdParty) string {
	var tmp []string
	for _, v := range list {
		tmp = append(tmp, fmt.Sprintf("{Name: %v, Data: %v}", strconv.Quote(v.Name), strconv.Quote(v.Data)))
	}
	return strings.Join(tmp, ",")
}

func GetNote(note string) string {
	return strconv.Quote(note)
}

// genOutPut 把生成产物写到显式输出目录(outPath / WithOutPath),而不是 cwd。
// 只有显式 WithDebug(true) 才会走到这里,默认不写任何文件,
// 生产只读文件系统、go test 都不会因为写工作目录而失败或改动被跟踪文件。
func genOutPut(outDir, modFile string) {
	_mu.Lock()
	defer _mu.Unlock()

	if len(outDir) == 0 {
		outDir = modFile + "/routers/"
	}

	if err := genCode(outDir, modFile); err != nil { // gen .go file
		mylog.Error(err)
	}

	_genInfo.Tm = time.Now().Unix()
	data, err := serializing.Encode(&_genInfo) // gob serialize 序列化
	if err != nil {
		mylog.Error(err)
		return
	}

	p := path.Join(outDir, "gen_router.data")
	if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
		mylog.Error(err)
		return
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		mylog.Error(err)
	}
}

func genCode(outDir, modFile string) error {
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
		return err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	// 用 go/format 替代 gofmt 子进程:
	// 不依赖 PATH、不需要 fork,也不会因为 Go 没装而静默不格式化。
	src := buf.Bytes()
	if formatted, ferr := format.Source(src); ferr != nil {
		mylog.Error(ferr) // 格式化失败不影响写文件,保留未格式化内容便于定位
	} else {
		src = formatted
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	return os.WriteFile(path.Join(outDir, "gen_router.go"), src, 0644)
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
	loadGenConf()

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
