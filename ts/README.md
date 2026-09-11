# @havocrao/picktui — TS/JS 绑定

薄绑定：只做「候选传入、结果取回」，渲染/按键/匹配全部在 Go 引擎
（`../go/cmd/picktui`）内完成，效果与 Go 宿主逐像素一致。
同份源码构建 esm + cjs + d.ts，TS 与 JS 共用。

## 安装

```console
$ npm install @havocrao/picktui
```

引擎发现顺序：`$PICKTUI_BIN` → `$PATH` 中的 `picktui`。交互 TUI 由引擎跑在
`/dev/tty`，本包 stdout 只回选中值。无 TTY 时引擎自动退化
（无 query 取首个、有 query 过滤取首），绑定无需分支；
取消（esc/ctrl+c）返回 `null`；协议错误抛出 `PicktuiError`（含 exitCode/stderr）。

## API

```ts
import { filter, resolve } from '@havocrao/picktui/filter'
import { pick, menu, rawPick } from '@havocrao/picktui/tui'
import { histGet, histSet } from '@havocrao/picktui/history'
import { confirmCheck, confirmAdd } from '@havocrao/picktui/confirm'
import type { Candidate, FilterOptions, FilteredCandidate, PickFlags } from '@havocrao/picktui/types'
import { engineVersion, assertEngineVersion } from '@havocrao/picktui'
```

| 函数 | 说明 |
|---|---|
| `filter(cands, query?, opts?)` | 过滤 + 高亮区间（ranges 为 rune 区间，空查询恒 `[]`） |
| `resolve(cands, query)` | 精确/唯一前缀解析；无唯一解返回 `null` |
| `pick(cands?, flags?)` | 交互过滤选择；`flags` 透传 `-q/--label/--sep/--fuzzy/-1/--auto` |
| `menu(label, cands)` | 多值缩写菜单（数字 1-9 直选） |
| `rawPick([...args])` | 透传引擎 pick 参数（`--from`/`--map` 等） |
| `histGet(label)` / `histSet(label, value)` | 选择记忆 |
| `confirmCheck` / `confirmAdd` | 自动匹配首次确认（add 幂等） |
| `assertEngineVersion(min?)` | 校验引擎版本不低于绑定声明的最低版本 |

## 开发

```console
$ npm install            # 安装 devDependencies（typescript）
$ npm run build          # tsc 双输出：dist/esm + dist/cjs + dist/types
$ npm test               # pretest 自动构建引擎二进制 → node:test 集成测试
$ npm run test:pack      # npm pack → 干净目录安装 → import/require 冒烟
```

测试均为对真实引擎二进制的 JSON 往返（无 TTY 依赖）。交互 TUI 路径
由引擎侧 model 级测试覆盖。