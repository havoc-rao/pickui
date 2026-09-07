# pickui

多语言可复用的 TUI 候选选择器：**Go 引擎（单一事实源）+ 稳定 JSON 协议 +
各语言薄绑定**。只要过滤、只要选择器、只要记忆 —— 取其一即可；
Go / TypeScript / JavaScript 宿主拿到**同样效果**。

fzf 风格的实时过滤选择器，源自 shr 的 `pick`/`_menu` 组件，抽离为独立可发布项目：
候选来源 → 过滤 → TUI 选择 → 输出选中值。

## How to use（引用包）

| 语言 | 安装 | 引用路径 |
|---|---|---|
| **Go**（引擎本体） | `go get github.com/havoc-rao/pickui/go@v0.1.0` | [pkg.go.dev/github.com/havoc-rao/pickui/go](https://pkg.go.dev/github.com/havoc-rao/pickui/go) → `import pickui "github.com/havoc-rao/pickui/go"` |
| **TS / JS**（薄绑定） | `npm install @havocrao/pickui` | [npmjs.com/package/@havocrao/pickui](https://www.npmjs.com/package/@havocrao/pickui) → `import { filter } from '@havocrao/pickui'` / `const { pick } = require('@havocrao/pickui')` |
| **引擎二进制**（任意宿主/脚本） | `go build -C go -o dist/pickui ./cmd/pickui` | [github.com/havoc-rao/pickui](https://github.com/havoc-rao/pickui) · [docs/protocol.md](docs/protocol.md) → `pickui pick/filter/resolve/hist/confirm` |

```go
// Go：过滤 / 选择器 / 记忆，全部函数直调
import pickui "github.com/havoc-rao/pickui/go"
hits := pickui.FilterStructuredCandidates(cands, "ele dev", pickui.FilterOptions{})
```

```ts
// TS：同一引擎的效果，薄绑定经 JSON 协议调用
import { filter, pick, histGet } from '@havocrao/pickui';
const hits = await filter(['electron:dev', 'electron:build'], 'ele dev'); // + 高亮区间
```

完整能力（filter / tui / history / confirm / 嵌入宿主对齐）见下方「快速开始」与
[go/README.md](go/README.md)、[ts/README.md](ts/README.md)。

## 仓库布局：每个语言包一个平级子目录

```
pickui/
├── go/                    ← Go 引擎包（模块 github.com/havoc-rao/pickui/go）
│   ├── cmd/pickui/        引擎二进制：协议子命令（pick/menu/filter/resolve/hist/confirm）
│   └── *.go               引擎库（package pickui）：filter / tui / history / confirm
├── ts/                    ← npm 包 @havocrao/pickui（TS + JS 双格式）
│   ├── src/               filter / tui / history / confirm / types（按模块 import，tree-shakable）
│   ├── test/              node:test 集成测试（对真实引擎二进制）
│   └── scripts/           引擎构建 / npm pack 可安装性验证
├── docs/
│   ├── plan.md            设计与迁移说明
│   └── protocol.md        协议 v1 规格（唯一契约）
├── Makefile               聚合入口：make test / test-go / test-ts / build-engine
└── README.md
```

**多语言平级原则**：每个语言包一个子目录，**Go 不占用根目录特殊位置**；将来加入
`py/`、`js/` 等新语言绑定，按同规格平级新增（见「多语言扩展约定」）。

- **Go 包**：`import "github.com/havoc-rao/pickui/go"` —— 引擎本体，函数直调（同进程）。
- **TS/JS 包**：`import { filter } from '@havocrao/pickui'` /
  `const { pick } = require('@havocrao/pickui')` —— 薄绑定，child_process + JSON 协议
  调引擎；同一份 dist 同时产出 esm（TS/ESM-import）与 cjs（JS/require）。
- **引擎二进制**：协议实现唯一权威，渲染/按键/匹配只存在于 Go 引擎；
  绑定层（或 shell 脚本）按 [docs/protocol.md](docs/protocol.md) 直接消费。
- 各包**独立管理版本、独立发布**：Go 模块按 git tag（`go/v0.1.0`）发布，npm 包按
  package.json 版本发布（`cd ts && npm publish`）；绑定可通过 `pickui version`
  校验引擎最低版本。

## 多语言扩展约定（新增语言绑定）

新增一个语言包（如 `py/`、`js/`）时按以下规格：

1. **平级子目录**：`<lang>/` 不嵌套、不占用根目录；目录内自包含（README、
   版本声明、源码、测试）。
2. **引擎发现**：`$PICKUI_BIN` → 平台 optionalDependencies 二进制包 → `PATH`
   （TS 绑定已实现此顺序，其他语言照抄）。
3. **协议即契约**：只消费 `docs/protocol.md`（子命令、JSON 字段、退出码 0/1/2/130）；
   禁止重实现渲染/匹配；`pickui version` 校验引擎最低版本。
4. **测试**：对真实引擎二进制的集成测试（JSON 往返），交互 TUI 由引擎侧
   model 级测试覆盖。
5. **发布独立**：各语言用各自生态的发布机制（Go tag / npm publish / PyPI …），
   版本号可独立演进，与引擎版本用「最低引擎版本」声明对齐。

## 快速开始

### Go（引擎 + 嵌入）

```console
$ cd go && go build ./cmd/pickui && go test ./...
```

```go
import pickui "github.com/havoc-rao/pickui/go"

// 只要过滤/高亮
hits := pickui.FilterStructuredCandidates(cands, "ele dev",
    pickui.FilterOptions{Mode: pickui.MatchSubstring})
ranges := pickui.HighlightRanges("electron:dev", "ele dev", opts)
v, ok := pickui.AutoResolve(cands, "re") // --auto 解析

// 只要交互选择器（调用方负责打开 /dev/tty）
res, err := pickui.Run(pickui.Options{Title: "pick", Cands: cands,
    FilterOpts: opts, StatusHint: "type to filter · enter select · esc cancel"}, tty)

// 只要选择记忆 / 首次确认
pickui.SavePick("npm run", "release")
pickui.LastPick(pickui.LoadPickHistory(), "npm run")
pickui.Confirm("npm run", "release")
pickui.IsConfirmed("npm run", "release")

// 完整 CLI 命令嵌入（宿主对齐：shr 即以此接入）
pickui.SetName("shr")               // 消息前缀/TUI 标题
pickui.SetOffEnv("SHR_PICK")        // 非交互开关环境变量
pickui.SetDataDir(core.ConfigDir()) // 记忆/确认文件目录
pickui.SetMapResolver(ResolvePickerMapArg) // --map 裸名解析钩子
os.Exit(pickui.Pick(args, os.Stdout, os.Stderr))
os.Exit(pickui.Menu(args, os.Stdout, os.Stderr))
```

详见 [go/README.md](go/README.md)。

### TS / JS（npm 包）

```console
$ cd ts && npm install && npm run build && npm test
```

```ts
import { filter, resolve, pick, histGet, histSet, confirmCheck, confirmAdd } from '@havocrao/pickui';
// 按模块 import（tree-shakable）：
// import { filter } from '@havocrao/pickui/filter'
// import { pick, menu } from '@havocrao/pickui/tui'

const hits = await filter(['electron:dev', 'electron:build', 'serve'], 'ele dev');
// [{ value: 'electron:dev', desc: '', ranges: [[0,3],[9,12]] }]

const v = await resolve(['build', 'release', 'serve'], 're'); // 'release' | null
const chosen = await pick(['alpha', 'beta'], { query: 'beta' });   // 交互 TUI（引擎渲染）
const chosen = await menu('db', ['mysql', 'postgres']);            // 数字 1-9 直选
await histSet('git p', 'pull'); await histGet('git p');            // 选择记忆
await confirmAdd('npm run', 'release'); await confirmCheck('npm run', 'release');
```

```js
// JS（CommonJS）同样直接调用：
const { pick, histGet } = require('@havocrao/pickui');
```

引擎发现顺序：`$PICKUI_BIN` → `$PATH` 中的 `pickui`。交互 TUI 由引擎跑在
`/dev/tty`，宿主 stdout 不被污染（`$(...)` 安全）。无 TTY 时引擎自动退化
（无 query 取首个、有 query 过滤取首），绑定无需分支。
详见 [ts/README.md](ts/README.md)。

### 引擎二进制（脚本直接消费）

```console
$ go build -C go -o dist/pickui ./cmd/pickui

$ git branch | dist/pickui pick                              # 交互过滤选择
$ PICKUI_PICK=off dist/pickui pick -q feat -1 <(git branch)  # 非交互确定性
$ dist/pickui _menu db mysql postgres                        # 菜单
$ printf 'dev\tDev\nbuild\tBuild\n' | dist/pickui filter --json --query de
$ printf 'build\nrelease\nserve\n' | dist/pickui resolve --json --query re
$ dist/pickui hist get "git p"; dist/pickui hist set "git p" pull
$ dist/pickui confirm check "npm run" release; dist/pickui confirm add "npm run" release
```

退出码：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消。数据走 stdout、消息走
stderr。完整协议见 [docs/protocol.md](docs/protocol.md)。

## 包与仓库的映射（import 直达对应子目录）

| 用户写 | 落到仓库 | 机制 |
|---|---|---|
| `import "github.com/havoc-rao/pickui/go"` | `go/` | Go 按 import path 找到仓库 `go/go.mod`（module `github.com/havoc-rao/pickui/go`，多模块仓库规范） |
| `import '@havocrao/pickui'` | `ts/` | npm 发布物即 `ts/` 目录内容；包名与目录解耦 |
| `import '@havocrao/pickui/tui'` | `ts/dist/esm/tui.js` | `package.json` `exports` 子路径映射（require → `dist/cjs/*.js`，types → `dist/types/*.d.ts`） |
| `PICKUI_BIN=/…/pickui` | 引擎二进制 | 跨语言消费引擎的唯一通道（TS 绑定/脚本），指向 `go/cmd/pickui` 构建产物 |

依赖方向只有一条：**ts/ → 引擎二进制（JSON 协议）**；ts/ 不依赖 go/ 的 Go
代码，两边靠 [docs/protocol.md](docs/protocol.md) 契约对齐，互不编译依赖。

## 版本与发布

| 包 | 位置 | 版本管理 |
|---|---|---|
| Go 引擎 | `go/`（github.com/havoc-rao/pickui/go） | git tag `go/v0.1.0`（多模块仓库前缀 tag，根 tag `v0.1.0` 亦可） |
| npm 包 | `ts/`（@havocrao/pickui） | package.json `version`（同号对齐，如 0.1.0） |

**本地开发（未发布）**：

```go
// Go 宿主（shr 等）：go.mod 加
replace github.com/havoc-rao/pickui/go => ../pickui/go
```

```jsonc
// TS/JS 宿主：package.json 加
{ "dependencies": { "@havocrao/pickui": "file:../pickui/ts" } }
// 任意宿主：把 go/cmd/pickui 构建的二进制放入 PATH，或设 PICKUI_BIN 指过去
```

**发布**：

```console
$ cd go && go test ./... -count=1 && cd ..          # Go 侧校验
$ git add -A && git commit && git tag go/v0.1.0 && git push --tags   # Go 发布
$ cd ts && npm test && npm run test:pack             # npm 侧校验（test:pack 验证装包后可调用）
$ npm publish                                        # npm 发布（tarball 仅含 dist/，零运行时依赖）
```

发布流程：全量测试绿（`go test ./...` + `npm test` + `npm run test:pack`）→ git tag →
`cd ts && npm publish`（需 npm 账号）。宿主发布后移除 replace / 改用 registry 版本。

## License

MIT