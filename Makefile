NAME := ginrpc

# 删除历史 `master` 目标里的 `go get -u ...@master`:
# 它会在构建期把依赖整体升级到未固定的 master 分支,不可复现且可能破坏构建。
# 依赖版本统一由 go.mod / go.sum 治理,这里只提供只读的构建/校验目标。
.PHONY: build test vet lint

build:
	go build ./...

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...
