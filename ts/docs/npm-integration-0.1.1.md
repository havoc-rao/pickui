# @havocrao/picktui 0.1.1 对接指南（npm）

> 面向 TS / JS 宿主的版本对接文档。0.1.1 起为**纯 TS 实现**：
> 零运行时依赖、无需引擎二进制、安装即用。
> 行为契约对齐 Go 引擎（`github.com/havoc-rao/picktui/go`），状态文件格式互兼容。

## 1. 版本说明

| 版本 | 形态 | 说明 |
|---|---|---|
| 0.1.0 | 薄绑定 | child_process + JSON 协议调 Go 引擎二进制（需 `PICKTUI_BIN`/PATH） |
| **0.1.1** | **纯 TS** | 匹配/高亮/交互/记忆全部在 JS 内实现，**移除引擎依赖** |

### 0.1.0 → 0.1.1 破坏性变更（升级前必读）

1. **移除**：`engineVersion()` / `assertEngineVersion()` / `MIN_ENGINE_VERSION` ——
   不再有引擎可查。用 `VERSION` 常量替代（本实现语义版本）。
2. **移除隐式 stdin 读取**：`pick()` 候选只来自参数；`pick([])` 抛
   `PicktuiError`（no candidates）。命令/管道来源改用
   `rawPick(['--from', '<cmd>', ...])`。
3. **行为保持**：API 签名（全 async）、`PicktuiError.exitCode` 语义（0/1/2/130）、
   取消返回 `null`、状态文件（history.toml/confirm.toml）格式与目录规则均不变，
   旧宿主大多只需删掉引擎版本校验代码即可升级。

## 2. 安装

```console
$ npm install @havocrao/picktui
```

- Node >= 18，零运行时依赖
- 无需引擎二进制、无需 `PICKTUI_BIN`、无需任何环境变量
- 同一份 dist 产出 ESM + CJS + d.ts；支持按模块 import（tree-shakable）

本地开发（未发布 / 用本地仓库）：

```jsonc
{ "dependencies": { "@havocrao/picktui": "file:../picktui/ts" } }
```

## 3. API 一览

```ts
import {
  filter, resolve, pick, menu, rawPick,
  histGet, histSet, confirmCheck, confirmAdd, VERSION,
} from '@havocrao/picktui';

// 按模块 import（tree-shakable）：
// import { filter, resolve } from '@havocrao/picktui/filter'
// import { pick, menu, rawPick } from '@havocrao/picktui/tui'
// import { histGet, histSet } from '@havocrao/picktui/history'
// import { confirmCheck, confirmAdd } from '@havocrao/picktui/confirm'
// import type { Candidate, FilterOptions, FilteredCandidate, PickFlags, PicktuiError } from '@havocrao/picktui/types'
```

| 函数 | 签名 | 说明 |
|---|---|---|
| `filter` | `(cands, query?, opts?) => Promise<FilteredCandidate[]>` | 过滤 + 高亮区间；保留输入顺序 |
| `resolve` | `(cands, query) => Promise<string \| null>` | 精确 → 唯一前缀解析；无唯一解 `null` |
| `pick` | `(cands?, flags?) => Promise<string \| null>` | 交互过滤选择（fzf 风格） |
| `menu` | `(label, cands) => Promise<string \| null>` | 多值缩写菜单（数字 1-9 直选） |
| `rawPick` | `(args) => Promise<string \| null>` | 透传 pick 参数（`--from`/`--map` 等） |
| `histGet` / `histSet` | `(label)` / `(label, value)` | 选择记忆（文件状态） |
| `confirmCheck` / `confirmAdd` | `(label, value) => Promise<boolean>` | 自动匹配首次确认 |
| `VERSION` | `string` | 本实现语义版本（0.1.1） |

类型：

- `Candidate { value: string; desc?: string }` —— desc 仅展示；过滤只匹配 key
- `FilterOptions { mode?: 'substring' \| 'token' \| 'fuzzy'; sep?: string }`
- `FilteredCandidate { value: string; desc: string; ranges: [number, number][] }`
- `PickFlags { query?, sep?, fuzzy?, label?, select1?, auto? }`
- `PicktuiError extends Error { exitCode: number; stderr: string }`

## 4. 快速对接示例

```ts
// ① 过滤 + 高亮（纯函数，headless 可用；ranges 为 rune 区间，空查询恒 []）
const hits = await filter(['electron:dev', 'electron:build', 'serve'], 'ele dev');
// [{ value: 'electron:dev', desc: '', ranges: [[0,3],[9,12]] }]

// ② 缩写解析
const v = await resolve(['build', 'release', 'serve'], 're'); // 'release' | null

// ③ 交互选择（染在 /dev/tty 或宿主 stdio，宿主 stdout 只回选中值）
const chosen = await pick(['alpha', 'beta'], { query: 'beta' }); // string | null

// ④ 菜单（数字 1-9 直选；label 同时是记忆键）
await menu('db', ['mysql', 'postgres']);

// ⑤ 命令来源 + 转换器
await rawPick(['--from', 'git branch', '--map', 'node ~/.config/pickers/git-branch.mjs', '--label', 'git co', '-1']);

// ⑥ 选择记忆 / 首次确认
await histSet('git p', 'pull');
await histGet('git p'); // 'pull'（空串 = 无记录）
await confirmAdd('npm run', 'release');   // 幂等
await confirmCheck('npm run', 'release'); // true
```

## 5. 关键语义

1. **stdout 干净**：POSIX 上交互 TUI 走 `/dev/tty`，宿主 stdout 只回选中值，
   `$(...)` 命令替换安全；无 `/dev/tty`（如 Windows）退化为宿主 stdio。
2. **无 TTY 自动退化**（无需分支）：`pick` 无 query 取首个；有 query 过滤取
   首个匹配，**无匹配抛错**（绝不静默回退首个）。
3. **取消 vs 错误**：esc/ctrl+c → `pick`/`menu`/`rawPick` 返回 `null`；
   运行/用法错误 → 抛 `PicktuiError`（`exitCode`：0 成功 / 1 运行错误 /
   2 用法错误 / 130 取消；`stderr` 含诊断文本）。
4. **状态目录**：`$PICKTUI_CONFIG_DIR` → `$XDG_CONFIG_HOME/picktui` →
   `~/.config/picktui`。`history.toml`/`confirm.toml` 与 Go 引擎同格式
   （旧 `picks.toml` 自动迁移），两实现可在同一目录共存。
5. **非交互开关**：`$PICKTUI_PICK=off` 时 `pick` 跳过 TUI（确定性脚本语义）。
6. **`--auto`**（pick flags.auto）：配合 query 精确/唯一前缀直接输出；首次匹配
   该 (label, value) 需经 `/dev/tty` 确认（记录 confirm）；`off`/无 tty/无 label
   跳过确认；无唯一解回退 TUI。
7. **候选来源**：`pick` 候选来自参数（不读宿主进程自身 stdin）；命令/管道来源
   用 `rawPick(['--from', ...])`（10s 超时；`--map` 转换器脚本缺失 fail-fast）。
8. **颜色降级**：设置 `NO_COLOR` 环境变量时 TUI 输出纯文本。

## 6. 升级清单（0.1.0 宿主）

- [ ] 删除 `assertEngineVersion()` / `engineVersion()` 调用
- [ ] 需要版本号判断处改用 `VERSION`
- [ ] 检查是否有 `pick([])` 依赖"读宿主 stdin 取候选"的用法 → 改
      `rawPick(['--from', ...])`
- [ ] 移除宿主环境里的 `PICKTUI_BIN` 设置（不再需要；残留不影响）
- [ ] 验证 `filter`/`pick` 返回值与存量断言一致（行为对齐，差分测试可复用）

## 7. 发布验证

```console
$ make npm-login       # 首次：登录官方 registry
$ make npm-dryrun      # 预览 tarball 内容（只应含 dist/）
$ make npm-publish     # test → test-pack → 登录预检 → publish → 打印线上版本
```

## 8. 参考

- 仓库：`github.com/havoc-rao/picktui`（`ts/` 为 npm 包源目录）
- 行为契约：`docs/integration/protocol.md`（Go 引擎同源语义）
- 包内文档：`ts/README.md`