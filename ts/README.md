# @havocrao/picktui — TUI 候选选择器（纯 TypeScript）

纯 TS 实现：过滤/匹配/交互选择/记忆全部在 JS 内完成，**零运行时依赖、
无需任何引擎二进制**，安装即用。行为与 Go 引擎（`../go/cmd/picktui`）语义
逐项对齐（匹配算法、高亮区间、非交互退化、状态文件格式互兼容）。
同份源码构建 esm + cjs + d.ts，TS 与 JS 共用。

## 安装

```console
$ npm install @havocrao/picktui
```

交互 TUI 由本包渲染：POSIX 上打开 `/dev/tty`（宿主 stdout 干净，`$(...)`
安全）；无 `/dev/tty` 的环境（如 Windows）退化为宿主 stdio。无 TTY 时
自动退化（无 query 取首个、有 query 过滤取首）；取消（esc/ctrl+c）
返回 `null`；错误抛 `PicktuiError`（含 exitCode/stderr 诊断）。

## API

```ts
import { filter, resolve } from '@havocrao/picktui/filter'
import { pick, menu, rawPick } from '@havocrao/picktui/tui'
import { histGet, histSet } from '@havocrao/picktui/history'
import { confirmCheck, confirmAdd } from '@havocrao/picktui/confirm'
import type { Candidate, FilterOptions, FilteredCandidate, PickFlags } from '@havocrao/picktui/types'
import { VERSION } from '@havocrao/picktui'
```

| 函数 | 说明 |
|---|---|
| `filter(cands, query?, opts?)` | 过滤 + 高亮区间（ranges 为 rune 区间，空查询恒 `[]`） |
| `resolve(cands, query)` | 精确/唯一前缀解析；无唯一解返回 `null` |
| `pick(cands?, flags?)` | 交互过滤选择；`flags` 透传 `-q/--label/--sep/--fuzzy/-1/--auto` |
| `menu(label, cands)` | 多值缩写菜单（数字 1-9 直选） |
| `rawPick([...args])` | 透传 pick 参数（`--from`/`--map` 等） |
| `histGet(label)` / `histSet(label, value)` | 选择记忆 |
| `confirmCheck` / `confirmAdd` | 自动匹配首次确认（add 幂等） |
| `VERSION` | 本实现语义版本 |

注：`pick` 候选来自参数（不读宿主进程自身 stdin）；需要命令/管道来源时用
`rawPick(['--from', '<cmd>', ...])`（10s 超时，失败带 stderr 诊断）。

## 开发

```console
$ npm install            # 安装 devDependencies（typescript，无运行时依赖）
$ npm run build          # tsc 双输出：dist/esm + dist/cjs + dist/types
$ npm test               # node:test 全量测试（filter/history/confirm/tui/model）
$ npm run test:pack      # npm pack → 干净目录安装 → import/require 冒烟
```

测试均为纯 JS 断言，无需二进制或 TTY：交互 TUI 的按键行为由 model 级测试覆盖。