# picktui — 聚合入口（每个语言包一个平级子目录）
#
# 用法：
#   make test          全部语言包测试（go test + npm test）
#   make test-go       仅 Go 引擎包
#   make test-ts       仅 TS/JS 绑定包（会自动构建引擎二进制）
#   make build-engine  构建引擎二进制到 dist/picktui
#   make test-pack     npm 打包可安装性验证（ts/）

.PHONY: test test-go test-ts build-engine test-pack

test: test-go test-ts

test-go:
	cd go && go test ./... -count=1

test-ts:
	cd ts && npm test

test-pack:
	cd ts && npm run test:pack

build-engine:
	cd go && go build -o ../dist/picktui ./cmd/picktui