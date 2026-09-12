# picktui TS 接入指南（@havocrao/picktui）

> 面向 TS / JS 宿主的对接文档：安装方式 + 基本用法 + 关键语义。
> 行为契约对齐 [protocol.md](./protocol.md)（纯 TS 实现，无需引擎二进制）。

## 1. 它是什么

`@havocrao/picktui` 是 **纯 TypeScript 实现**的 TUI 候选选择器：过滤匹配、
高亮区间、交互选择（fzf 风格）、选择记忆、首次确认全部在 JS 内完成，
**零运行时依赖、不依赖任何外部二进制**。语义与 Go 引擎
（`github.com/havoc-rao/picktui/go`）逐项对齐，状态文件（history.toml /
confirm.toml）格式互兼容，可在同一数据目录共存使用。

- npm：`@havocrao/picktui`（当前 0.1.1）
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

无需引擎二进制、无需设置任何环境变量，安装即用。

## 3. 基本用法

```ts
import {
  filter, resolve, pick, menu, rawPick,
  histGet, histSet, confirmCheck, confirmAdd, VERSION,
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

// ③ 交互过滤选择（fzf 风格；POSIX 上渲染在 /dev/tty，宿主 stdout 只回选中值）
const chosen = await pick(['alpha', 'beta'], { query: 'beta' }); // string | null

// ④ 多值缩写菜单（数字 1-9 直选）
await menu('db', ['mysql', 'postgres']);

// ⑤ 未包装的 pick 参数（--from/--map 等）
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
| `pick` | `(cands?, flags?) => Promise<string \| null>` | 交互过滤选择；候选来自参数（不读宿主 stdin） |
| `menu` | `(label, cands) => Promise<string \| null>` | 数字直选菜单 |
| `rawPick` | `(args) => Promise<string \| null>` | 透传 pick 参数（`--from`/`--map` 等） |
| `histGet` / `histSet` | `(label) => Promise<string>` / `(label, value) => Promise<void>` | 选择记忆 |
| `confirmCheck` / `confirmAdd` | `(label, value) => Promise<boolean>` | 自动匹配首次确认（add 幂等） |
| `VERSION` | `string` | 本实现语义版本 |

类型：

- `Candidate { value: string; desc?: string }` — desc 仅展示；过滤只匹配 key
- `FilterOptions { mode?: 'substring' | 'token' | 'fuzzy'; sep?: string }` — sep 仅 token 模式（默认 `"_"`）
- `FilteredCandidate { value: string; desc: string; ranges: [number, number][] }`
- `PickFlags { query?, sep?, fuzzy?, label?, select1?, auto? }`
- `PicktuiError extends Error { exitCode: number; stderr: string }`

## 5. 关键语义（接入前必读）

1. **stdout 干净**：POSIX 上交互 TUI 走 `/dev/tty`，宿主 stdout 只回选中值，
   `$(...)` / 管道捕获安全；无 `/dev/tty` 的环境（如 Windows）退化为宿主 stdio。
2. **无 TTY 自动退化**（无需分支）：`pick` 无 query 取首个候选；
   有 query 过滤取首个匹配，**无匹配报错**（绝不静默回退）。
3. **取消 vs 错误**：用户取消（esc/ctrl+c）→ `pick`/`menu`/`rawPick` 返回
   `null`（不是异常）；运行/用法错误抛 `PicktuiError`（含 `exitCode`/`stderr`）。
4. **退出码语义**：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消
   （`PicktuiError.exitCode` 携带）。
5. **状态目录**（hist/confirm）：`$PICKTUI_CONFIG_DIR` → `$XDG_CONFIG_HOME/picktui`
   → `~/.config/picktui`，与 Go 引擎同一份文件（格式互兼容）。
   非交互开关：`$PICKTUI_PICK=off` 时 pick 跳过 TUI（确定性脚本语义）。
6. **纯函数面**：`filter` / `resolve` 无 TTY 依赖，可在 headless / 服务端场景使用。
7. **`--auto`**（pick flags.auto）：配合 query 精确/唯一前缀直接输出；首次匹配
   该 (label, value) 经 `/dev/tty` 确认；无 tty / `PICKTUI_PICK=off` / 无 label
   跳过确认；无唯一解回退 TUI。
8. **候选来源**：`pick` 候选来自参数；命令/管道来源用 `rawPick(['--from', ...])`
   （10s 超时）。本包不读取宿主进程自身的 stdin。

## 6. 参考

- 仓库：[github.com/havoc-rao/picktui](https://github.com/havoc-rao/picktui)
- 行为契约：`docs/integration/protocol.md`（Go 引擎同源语义）
- 绑定文档：`ts/README.md`；绑定源码：`ts/src/`