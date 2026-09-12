# picktui — 聚合入口（每个语言包一个平级子目录）
#
# 用法：
#   make test             全部语言包测试（go test + npm test）
#   make test-go          仅 Go 引擎包
#   make test-ts          仅 TS 包（纯 TS 实现，无引擎依赖）
#   make test-pack        npm 打包可安装性验证（build + 干净目录装包冒烟）
#   make build-engine     构建引擎二进制到 dist/picktui
#   make npm-login        登录官方 npm registry（发布前一次性操作）
#   make npm-dryrun       npm publish --dry-run 预览包内容（不落 registry）
#   make npm-publish      官方源发布 @havocrao/picktui（自动预检：测试 + 登录）

.PHONY: test test-go test-ts test-pack build-engine npm-login npm-dryrun npm-publish

# 官方 npm registry（发布专用；日常 npm install 走镜像不受影响）
NPM_REGISTRY := https://registry.npmjs.org/

test: test-go test-ts

test-go:
	cd go && go test ./... -count=1

test-ts:
	cd ts && npm test

test-pack:
	cd ts && npm run test:pack

build-engine:
	cd go && go build -o ../dist/picktui ./cmd/picktui

npm-login:
	cd ts && npm login --registry=$(NPM_REGISTRY)

npm-dryrun:
	cd ts && npm publish --dry-run --registry=$(NPM_REGISTRY)

# 发布前预检：全量测试 + 打包冒烟 + 已登录官方源；发布走 prepack 自动构建
npm-publish: test-pack
	@npm whoami --registry=$(NPM_REGISTRY) >/dev/null 2>&1 || \
	  (echo "错误：未登录官方 npm，请先执行 make npm-login" && exit 1)
	cd ts && npm publish --registry=$(NPM_REGISTRY)
	@npm view @havocrao/picktui version