package ginrpc

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"runtime"
	"strings"

	"github.com/xxjwxc/public/errors"
	"github.com/xxjwxc/public/tools"
)

// find and get module info , return module [ name ,path ]
func getModuleInfo() (string, string) {
	index := 2
	// This is used to support third-party package encapsulation
	// 这样做用于支持第三方包封装,(主要找到main调用者)
	for true { // find main file
		_, filename, _, ok := runtime.Caller(index)
		if ok {
			if strings.HasSuffix(filename, "runtime/asm_amd64.s") {
				index = index - 2
				break
			}
			index++
		} else {
			panic(errors.New("package parsing failed:can not find main files"))
		}
	}

	_, filename, _, _ := runtime.Caller(index)
	filename = strings.Replace(filename, "\\", "/", -1) // offset
	for true {
		n := strings.LastIndex(filename, "/")
		if n > 0 {
			filename = filename[0:n]
			if tools.CheckFileIsExist(filename + "/go.mod") {
				list := tools.ReadFile(filename + "/go.mod")
				if len(list) > 0 {
					line := strings.TrimSpace(list[0])
					if len(line) > 0 && strings.HasPrefix(line, "module") { // find it
						return strings.TrimSpace(line[7:]), filename
					}
				}
			}
		} else {
			panic(errors.New("package parsing failed:can not find module file[go.mod] , golang version must up 1.11"))
		}
	}

	// never reach
	return "", ""
}

// Return to relative path . 通过module 游标返回包相对路径
func evalSymlinks(modPkg, modFile, objPkg string) string {
	if strings.EqualFold(objPkg, "main") { // if main return default path
		return modFile
	}

	if strings.HasPrefix(objPkg, modPkg) {
		return modFile + strings.Replace(objPkg[len(modPkg):], ".", "/", -1)
	}

	// get the error space
	panic(errors.Errorf("can not eval pkg:[%v] must include [%v]", objPkg, modPkg))
}

// getAstPkgs Parsing source file ast structure (with objname restriction).解析源文件ast结构(带objName限制)
func getAstPkgs(objPkg, objFile, objName string) (*ast.Package, bool) {
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, objFile, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)
	if err != nil {
		return nil, false
	}

	// check the package is same.判断 package 是否一致
	for _, pkg := range astPkgs {
		if objPkg == pkg.Name { // find it
			return pkg, true
		}
	}

	// not find . maybe is main pakge and find main package
	if objPkg == "main" {
		dirs := tools.GetPathDirs(objFile) // get all of dir
		for _, dir := range dirs {
			if !strings.HasPrefix(dir, ".") {
				pkg, b := getAstPkgs(objPkg, objFile+"/"+dir, objName)
				if b {
					return pkg, true
				}
			}
		}
	}

	// ast.Print(fileSet, astPkgs)

	return nil, false
}
