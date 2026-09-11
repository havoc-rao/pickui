# picktui TS 接入指南（@havocrao/picktui）

> 面向 TS / JS 宿主的对接文档：安装方式 + 基本用法 + 关键语义。
> 协议唯一契约：[protocol.md](./protocol.md)。

## 1. 它是什么

picktui = **Go 引擎（单一事实源）+ 稳定 JSON 协议 v1 + 各语言薄绑定**。
TS 包 `@havocrao/picktui` 是薄绑定：只做「候选传入、结果取回」
（child_process + JSON 协议），渲染/按键/匹配全部在 Go 引擎二进制
（`picktui`）内完成，效果与 Go 宿主逐像素一致。

- npm：`@havocrao/picktui`（当前 0.1.0，已发布；旧名 `@havocrao/pickui@0.1.0` 已弃用，勿再依赖）
- 产物：ESM + CJS 双格式 + d.ts，零运行时依赖，Node >= 18
- 支持按模块 import（tree-shakable）

## 2. 安装

```console
$ npm install @havocrao/picktui
```

本地开发（未发布 / 用本地仓库时）：

```jsonc
// 宿主 package.json
{ "dependencies": { "@havocrao/picktui": "file:../picktui/ts" } }
```

### 引擎前提（必需）

绑定**不含引擎二进制**。引擎发现顺序：`$PICKTUI_BIN` → `$PATH` 中的 `picktui`。

```console
# 构建引擎（Go 仓库 github.com/havoc-rao/picktui）
git clone https://github.com/havoc-rao/picktui.git
go build -C go -o picktui ./cmd/picktui
export PICKTUI_BIN=/abs/path/to/picktui    # 或将 picktui 放入 PATH
picktui version                           # 应输出：picktui 0.1.0
```

绑定声明最低引擎版本 `MIN_ENGINE_VERSION = '0.1.0'`，宿主启动时可校验：

```ts
import { assertEngineVersion } from '@havocrao/picktui';
await assertEngineVersion(); // 不满足抛 PicktuiError
```

## 3. 基本用法

```ts
import {
  filter, resolve, pick, menu, rawPick,
  histGet, histSet, confirmCheck, confirmAdd,
} from '@havocrao/picktui';

// 按模块 import（tree-shakable）：
// import { filter } from '@havocrao/picktui/filter'
// import { pick, menu, rawPick } from '@havocrao/picktui/tui'
// import { histGet, histSet } from '@havocrao/picktui/history'
// import { confirmCheck, confirmAdd } from '@havocrao/picktui/confirm'
// import type { Candidate, FilterOptions, FilteredCandidate, PickFlags, PicktuiError } from '@havocrao/picktui/types'

// ① 过滤 + 高亮（纯函数，非交互；ranges 为 [start,end) rune 区间，空查询恒 []）
const hits = await filter(['electron:dev', 'electron:build', 'serve'], 'ele dev');
// → [{ value: 'electron:dev', desc: '', ranges: [[0,3],[9,12]] }]

// ② 缩写解析：精确（大小写敏感）→ 大小写不敏感唯一前缀；无唯一解返回 null
const v = await resolve(['build', 'release', 'serve'], 're'); // 'release' | null

// ③ 交互过滤选择（fzf 风格；引擎在 /dev/tty 渲染，宿主 stdout 只回选中值）
const chosen = await pick(['alpha', 'beta'], { query: 'beta' }); // string | null

// ④ 多值缩写菜单（数字 1-9 直选）
await menu('db', ['mysql', 'postgres']);

// ⑤ 引擎未包装的 pick 参数（--from/--map 等）
await rawPick(['--from', 'git branch', '--label', 'git co', '-1']);

// ⑥ 选择记忆 / 首次确认
await histSet('git p', 'pull');
await histGet('git p'); // 'pull'（空串 = 无记录）
await confirmAdd('npm run', 'release');            // 幂等
await confirmCheck('npm run', 'release');          // true
```

JS（CommonJS）同样直接调用：

```js
const { pick, histGet } = require('@havocrao/picktui');
```

## 4. API 速查

| 函数 | 签名 | 说明 |
|---|---|---|
| `filter` | `(cands, query?, opts?) => Promise<FilteredCandidate[]>` | 过滤 + 高亮区间；保留输入顺序 |
| `resolve` | `(cands, query, opts?) => Promise<string \| null>` | 唯一解解析；无唯一解 `null` |
| `pick` | `(cands?, flags?) => Promise<string \| null>` | 交互过滤选择；候选经 stdin 传入 |
| `menu` | `(label, cands) => Promise<string \| null>` | 数字直选菜单；候选走位置参数（注意 argv 长度） |
| `rawPick` | `(args) => Promise<string \| null>` | 透传引擎 pick 参数 |
| `histGet` / `histSet` | `(label) => Promise<string>` / `(label, value) => Promise<void>` | 选择记忆 |
| `confirmCheck` / `confirmAdd` | `(label, value) => Promise<boolean>` | 自动匹配首次确认（add 幂等） |
| `engineVersion` / `assertEngineVersion` | `() => Promise<string>` / `(min?) => Promise<string>` | 引擎版本读取 / 校验 |

类型：

- `Candidate { value: string; desc?: string }` — desc 仅展示；序列化 `key<TAB>description`，过滤只匹配 key
- `FilterOptions { mode?: 'substring' | 'token' | 'fuzzy'; sep?: string }` — sep 仅 token 模式（默认 `"_"`）
- `FilteredCandidate { value: string; desc: string; ranges: [number, number][] }`
- `PickFlags { query?, sep?, fuzzy?, label?, select1?, auto? }`
- `PicktuiError extends Error { exitCode: number; stderr: string }`

## 5. 关键语义（接入前必读）

1. **stdout 干净**：交互 TUI 走引擎的 `/dev/tty`，宿主 stdout 只回选中值，
   `$(...)` / 管道捕获安全。
2. **无 TTY 自动退化**（绑定无需分支）：`pick` 无 query 取首个候选；
   有 query 过滤取首个匹配，**无匹配报错退出 1**（绝不静默回退）。
3. **取消 vs 错误**：用户取消（esc/ctrl+c，退出码 130）→ `pick`/`menu`/`rawPick`
   返回 `null`（不是异常）；运行/协议错误抛 `PicktuiError`（含 `exitCode`/`stderr`）。
4. **退出码协议**：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消。
   数据走 stdout，消息走 stderr。
5. **状态目录**（hist/confirm）：`$PICKTUI_CONFIG_DIR` → `$XDG_CONFIG_HOME/picktui`
   → `~/.config/picktui`。非交互开关：`$PICKTUI_PICK=off` 时 pick 跳过 TUI
   （确定性脚本语义）。
6. **纯函数面**：`filter` / `resolve` 无 TTY 依赖，可在 headless / 服务端场景使用。
7. **`--auto`**（pick flags.auto）：配合 query 精确/唯一前缀直接输出；首次匹配
   该 (label, value) 经 `/dev/tty` 确认；无 tty / `PICKTUI_PICK=off` / 无 label
   跳过确认；无唯一解回退 TUI。

## 6. 参考

- 仓库：[github.com/havoc-rao/picktui](https://github.com/havoc-rao/picktui)
- 协议契约（唯一事实源）：`docs/integration/protocol.md`
- 绑定文档：`ts/README.md`；绑定源码：`ts/src/`
