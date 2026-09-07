# pickui 抽取计划（docs/plan）— 多语言可复用的 TUI 候选选择器

> 目标：把 shr 的「TUI 辅助选择执行 CLI」整理为独立项目 **pickui**——
> Go 引擎（单一事实源）+ 稳定 JSON 协议 + 各语言薄绑定（Go 直嵌 / TS / JS），
> 其他项目**引入包、按需食用**：只要过滤、只要选择器、只要记忆，取其一即可；
> Go / TypeScript / JavaScript 宿主得到**同样效果**。

## 1. 背景与定位

shr 的 `pick` / `_menu` / `tui` / 过滤引擎是一块自洽、与规则树零耦合的功能：
候选来源 → 过滤 → TUI 选择 → 输出选中值。本次把它抽为独立项目 **pickui** 并发布。

与纯 Go 库方案的关键差异：TUI 是终端渲染问题，无法用「三个语言各写一套」保证
同样效果。因此 pickui 采用**引擎 + 协议 + 绑定**三层架构：

```
┌─ 宿主项目 ────────────────────────────────────────────┐
│  Go 项目：import github.com/havoc-rao/pickui/go      │
│    ├─ filter（纯函数）   tui（选择器）  history/confirm  │
│    └─ Pick/Menu（完整 CLI 命令嵌入）                    │
│  TS/JS 项目：npm 包 @havocrao/pickui                  │
│    ├─ filter()           pick()          history()     │
│    └─ 按模块 import，tree-shakable，效果与 Go 一致       │
└──────────────┬──────────────────────────┬──────────────┘
               │ 函数直调（同进程）          │ child_process + 协议
               ▼                          ▼
   ┌─────────────────────────────────────────────────────┐
   │ pickui 引擎（Go 单二进制，无运行时依赖，权威实现）       │
   │  pick / menu / filter / resolve / hist / confirm     │
   │  交互走 /dev/tty；数据走 stdin ↔ JSON ↔ stdout；      │
   │  退出码协议 0/1/2/130                                │
   └─────────────────────────────────────────────────────┘
```

- **为什么交互走引擎**：渲染、按键、过滤算法只有一份（引擎），任何语言宿主拿到的
  视觉与交互效果逐像素一致；绑定层只做「候选传入、结果取回」，薄而稳。
- **为什么协议是 CLI 子命令 + JSON 而非长连接 RPC**：选择是瞬态动作（打开→选→关），
  无状态子命令最简、可脚本化、CI 可测；历史/确认是文件状态，由引擎子命令读写。

## 2. 本次范围（M1：Go 引擎 + 协议子命令 + shr 切换）

> 仓库布局：多包 monorepo，**每个语言包一个平级子目录，Go 不占用根目录**——
> `go/`（Go 引擎模块 `github.com/havoc-rao/pickui/go`，含引擎库与 `cmd/pickui`
> 二进制）、`ts/`（npm 包 `@havocrao/pickui`，esm+cjs+d.ts 双格式同时服务
> TS 与 JS）、`docs/`。根目录只放仓库级内容（README/LICENSE/docs/CI/聚合
> Makefile）。各包独立管理版本、独立发布（Go 前缀 tag `go/vX.Y.Z` / npm
> version）；将来 `py/`、`js/` 等新语言绑定按「多语言扩展约定」平级加入。

### 2.1 移入 pickui（从 shr 复制并泛化）

| shr 源文件 | pickui 落点（均在 `go/` 子目录） | 内容 |
|---|---|---|
| `tui/tui.go` | `tui.go` | bubbletea 过滤选择器（Options/Result/Run/model） |
| `core/filter.go` | `filter.go` + `cand.go` | 匹配引擎（子串 AND/token 前缀/模糊）、高亮、AutoResolve；候选来源与结构化解析、MapperScriptError |
| `core/history.go` | `history.go` | 选择记忆（history.toml，含旧 picks.toml 迁移） |
| `core/confirm.go` | `confirm.go` | 自动匹配首次确认（confirm.toml） |
| `cli/pick.go` | `pick.go` | `Pick(args, out, errw) int` 完整命令逻辑 |
| `cli/menu.go` | `menu.go` | `Menu(args, out, errw) int` 完整命令逻辑 |
| 新增 | `cmd/filter.go` 等 | 协议子命令（见 §3） |
| 对应测试 | `*_test.go` | tui/filter/history/confirm/pick/协议 测试 |

### 2.2 留在 shr

规则树（`core/match.go`/`trie.go`/`config.go`/`shellgen.go`/`consts.go` 大部分）、
shr 专属命令（rules/list/init/setup/pickers/misc/update/completion/help/spec）、
`cli.go` 输出着色体系。shr 的 `pick`/`_menu` 保留为薄包装。

## 3. 引擎协议 v1（Go 二进制子命令，TS/JS 绑定与脚本共用的稳定接口）

| 子命令 | 输入 | 输出 | 无 TTY 可用 |
|---|---|---|---|
| `pick [flags]` | 候选（位置参数/stdin/--from/--map）+ `-q/--sep/--fuzzy/--label/--auto/-1` | stdout 选中值；0/1/2/130 | 退化：无 query 取首个、有 query 过滤取首 |
| `menu <label> <cand...>` | 同上 | 同上 | 退化：取首个 + stderr 警告 |
| `filter --json` | stdin 候选行 + `--query/--mode/--sep` | JSON：`[{value,desc,ranges}]` | ⭕ 纯函数 |
| `resolve --json` | 同上 | JSON：`{value}` 或空 + 退出码 1 | ⭕ 纯函数 |
| `hist get <label>` / `hist set <label> <value>` | args | JSON / 空 | ⭕ 文件状态 |
| `confirm check <label> <value>` / `confirm add <label> <value>` | args | JSON：`{confirmed:bool}` | ⭕ 文件状态 |

- `--mode`：`substring`（默认）/ `token` / `fuzzy`；`--sep` 配合 token。
- `pick/menu` 同时是 shr 的现有命令语义（原样保留）；filter/resolve/hist/confirm
  是新增的纯数据面，供 TS/JS 绑定与 shell 脚本直接消费。
- 协议细节（字段、示例、错误约定）单独维护 `docs/protocol.md`。

## 4. 语言绑定设计（按需食用）

### 4.1 Go 绑定 = 引擎本体

`github.com/havoc-rao/pickui/go` 包内按需取用，无进程开销：

| 需求 | API |
|---|---|
| 只要过滤/高亮 | `FilterStructuredCandidates` / `HighlightRanges` / `AutoResolve` |
| 只要交互选择器 | `tui.Run(opts, tty)` / `FilterOptions` / `Candidate` |
| 只要选择记忆/确认 | `SavePick/LastPick/LoadPickHistory` / `Confirm/IsConfirmed` |
| 完整 CLI 命令嵌入 | `Pick(args, out, errw)` / `Menu(args, out, errw)` |
| 宿主对齐 | `SetName` / `SetOffEnv` / `SetDataDir` / `SetMapResolver` |

shr 即按 4.1 接入：`SetName("shr")`、`SetOffEnv("SHR_PICK")`、
`SetDataDir(core.ConfigDir())`、`SetMapResolver(ResolvePickerMapArg)`，
消息文本、记忆文件位置、环境变量逐项不变（详细步骤见 §6）。

### 4.2 TS / JS 绑定（M2，本 plan 预留设计）

npm 包 `@havocrao/pickui`（esm + cjs 双输出，模块按需 import）：

| 模块 | 能力 | 实现 |
|---|---|---|
| `filter` | 过滤 + 高亮区间 + AutoResolve | 调引擎 `filter/resolve --json`；可选内嵌同构 JS 实现（`PICKUI_NO_BIN=1` 时启用，纯数据无 TUI） |
| `tui` | 交互选择 / 菜单 | 引擎跑 TUI，`child_process` 收 stdout 选中值，stderr 透传 |
| `history` / `confirm` | 记忆与确认 | 调引擎 `hist/confirm` 子命令 |
| `types` | `Candidate/Result/FilterOptions` 类型 | 纯类型，零依赖 |

- 引擎发现顺序：`$PICKUI_BIN` → 平台 optionalDependencies 二进制包 → `PATH`。
- TTY 语义与引擎一致：交互渲染走 `/dev/tty`，宿主 stdout 不被污染（`$(...)` 安全）。
- 效果一致性由引擎保证，绑定层零渲染逻辑。

### 4.3 按需食用矩阵

| 需求 | Go | TS/JS |
|---|---|---|
| 只要过滤/高亮 | `FilterStructuredCandidates` | `import { filter } from '@havocrao/pickui/filter'` |
| 只要交互选择器 | `tui.Run` | `pick()` / `menu()` |
| 只要选择记忆 | `SavePick/LastPick` | `history.get/set` |
| 只要 --auto 解析 | `AutoResolve` | `resolve()` |
| 完整命令嵌入 | `Pick/Menu` | `runPick args`（透传引擎） |

## 5. 泛化点（shr 硬编码 → pickui 可配置）

| shr 硬编码 | pickui 泛化 |
|---|---|
| 命令前缀 `shr `/`shr:` | `SetName()`，默认 `pickui` |
| 环境变量 `SHR_PICK=off` | `SetOffEnv()`，默认 `PICKUI_PICK` |
| 数据目录 `~/.config/shr` | `SetDataDir()`，默认 `$XDG_CONFIG_HOME/pickui` |
| `--map` 裸名解析（shr pickers） | `SetMapResolver()` 钩子 |
| 输出直写 os.Stdout/os.Stderr | `Pick/Menu` 接收 `io.Writer`（shr 传 paintedWriter 保留着色） |
| 退出码 | 不变：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消 |

## 6. shr 改造步骤（M1 后半）

1. 删除：`tui/`、`core/filter.go`、`core/history.go`、`core/confirm.go`、
   `cli/pick.go`、`cli/menu.go` 及对应测试；`core/consts.go` 移除随迁常量。
2. `cli/cli.go` 增加 `paintedWriter`（逐行着色包装）与 `init()` 四项对齐调用（§5）。
3. 新建 `cli/pick.go`/`cli/menu.go` 薄包装（menu 保留 `helpText.Menu` 用法错误）。
4. `cli/setup.go`、`cli/pickers.go` 改 import：`core.Candidate → pickui.Candidate`、
   `core.FilterOptions → pickui.FilterOptions`、`tui.Run → pickui.Run`。
5. `tests/pickers_test.go` 依赖的 `shrConfigDir` 随迁走 → shr 侧新增
   `tests/helper_test.go` 保留。
6. `go.mod`（shr 侧）：`require github.com/havoc-rao/pickui/go v0.1.0` +
   `replace => ../pickui/go`（发布后移除 replace）。
7. 验收：`go build ./... && go test ./... -count=1` 全绿；冒烟 `shr pick`（交互+非交互）
   与 `shr _menu`。

## 7. 测试策略

- pickui 独立测试：迁移 + 适配现有全部用例（filter 匹配矩阵、TUI 按键行为、
  选择记忆与迁移、确认、ParseConfirmAnswer），新增 DataDir 解析与**协议子命令
  测试**（filter/resolve/hist/confirm 的 JSON 往返，无 TTY 依赖）。
- shr 侧既有测试为回归基线：pickers_test（钩子链）、picks_shellgen_test
  （`__picks` 注入调 `shr pick` 字符串）、rc_install_test（真实构建二进制）。

## 8. 发布（M1 → M2）

1. 用户创建 GitHub 空仓库 `havoc-rao/pickui`（不要 README/license）。
2. 提交（含 docs/plan.md、docs/protocol.md）→ push → 打 `v0.1.0` tag。
3. shr 移除 replace，依赖 `github.com/havoc-rao/pickui/go v0.1.0`，全量测试后单独 commit。
4. M2：TS/JS 绑定入库（`ts/`），`npm publish`（需 npm 账号/token，
   用户执行或提供；包名 `@havocrao/pickui`）。
5. 后续可选：GoReleaser 交叉编译 + npm 平台 optionalDependencies 二进制包，
   `PICKUI_BIN` 自动探测完善。

## 9. 风险与对策

| 风险 | 对策 |
|---|---|
| 消息文本/行为变化破坏 shr 既有脚本 | SetName 对齐 + paintedWriter 保留着色；shr 全量测试回归 |
| 记忆文件位置变化导致用户记忆丢失 | shr 显式 SetDataDir(core.ConfigDir()) |
| replace 指向本地路径导致 CI 构建失败 | 发布 tag 后立即移除 replace 并提交 |
| TS 绑定与引擎版本错配 | 协议文档化 + 绑定声明最低引擎版本，`pickui --version` 校验 |
| TUI 效果跨语言不一致 | 渲染/按键/过滤唯一实现在引擎；绑定层禁止自绘或重实现匹配 |
| 双仓库/多包后续开发不同步 | 引擎改动统一进 pickui；shr 只保留薄包装 |