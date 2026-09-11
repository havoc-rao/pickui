# picktui 引擎协议 v1

> 稳定接口：Go 二进制子命令 + JSON。TS/JS 绑定与 shell 脚本共用；
> 绑定层禁止自绘或重实现匹配，效果一致性由引擎保证。

- 引擎：`picktui` 单二进制（Go 编译，无运行时依赖，权威实现）。
- 退出码协议：**0** 成功 / **1** 运行错误（含 resolve 无唯一解）/ **2** 用法错误 /
  **130** 取消（用户 esc/ctrl+c）。
- 数据约定：**stdout 只承载数据**（选中值或 JSON），**stderr 承载消息**（错误/警告）。
  因此 `$(picktui pick ...)`、`picktui filter --json | jq ...` 输出永不污染。
- 错误约定：运行错误 → stderr 纯文本 + 退出码 1；用法错误 → stderr 用法说明 + 退出码 2；
  错误时 stdout 为空或不承诺内容（绑定层不应解析）。
- 交互约定：TUI 渲染与按键输入走 `/dev/tty`，不经 stdout——
  `picktui --version`（或 `version`）可校验最低引擎版本（绑定层声明要求）。

## 子命令总览

| 子命令 | 输入 | 输出 | 无 TTY 可用 |
|---|---|---|---|
| `pick [flags]` | 候选（位置参数/stdin/`--from`/`--map`）+ `-q/--sep/--fuzzy/--label/--auto/-1` | stdout 选中值；0/1/2/130 | 退化：无 query 取首个、有 query 过滤取首 |
| `menu <label> <cand...>` | 同 pick | 同 pick | 退化：取首个 + stderr 警告 |
| `filter --json` | stdin 候选行 + `--query/--mode/--sep` | JSON：`[{value,desc,ranges}]` | ⭕ 纯函数 |
| `resolve --json` | 同 filter | JSON：`{value}` 或空 + 退出码 1 | ⭕ 纯函数 |
| `hist get <label>` / `hist set <label> <value>` | args | JSON / 空 | ⭕ 文件状态 |
| `confirm check <label> <value>` / `confirm add <label> <value>` | args | JSON：`{confirmed:bool}` | ⭕ 文件状态 |

- `--mode`：`substring`（默认）/ `token` / `fuzzy`；`--sep` 配合 token
  （分隔符集合，默认 `_`，可多字符如 `:_`）。
- 结构化候选约定：源输出每行按 `key<TAB>description` 拆分为"选中值 + 展示描述"；
  无 TAB 的行整行作为 key；key 清洗：trim → 去 `* ` / `+ ` 前缀（git branch 标记）
  → 去重保序。**过滤只匹配 key，描述不参与。**
- `hist`/`confirm` 状态文件位于数据目录（见下），与 `pick --label` / `--auto`
  命令的 TUI 记忆、首次确认共用同一份。

## 数据目录与环境

数据目录解析优先级（`hist`/`confirm`/`pick --label` 状态文件所在）：

1. `$PICKTUI_CONFIG_DIR`（显式覆盖）
2. `$XDG_CONFIG_HOME/picktui`
3. `~/.config/picktui`

状态文件：`history.toml`（选择记忆，含旧 `picks.toml` 自动迁移）、`confirm.toml`
（首次确认记录）。非交互开关：`$PICKTUI_PICK=off` 时 `pick` 跳过 TUI
（确定性的脚本语义），`--auto` 同时跳过首次确认询问。

## pick

```
picktui pick [flags] [cand...]
git branch | picktui pick
picktui pick --from "git branch"
picktui pick --from 'npm run' --map 'node .../npm-run.mjs' --label 'npm run' --auto -q re
```

| flag | 含义 |
|---|---|
| `-q, --query <text>` | 预填查询；非 TTY 时过滤后打印首个匹配 |
| `--from <cmd>` | 候选来源：执行命令取 stdout（10s 超时，stderr 并入错误） |
| `--map <cmd>` | 转换器：接收 `--from`（或 stdin）的原始输出到自身 stdin，输出结构化行 |
| `--sep <chars>` | token 前缀模式的分隔符集合 |
| `--fuzzy` | 子序列模糊匹配 |
| `--label <name>` | 选择记忆键（history.toml）与 TUI 副标题 |
| `-1, --select-1` | 仅一个匹配时自动选中（跳过 TUI） |
| `--auto` | 配合 `-q`：精确或唯一前缀匹配直接输出；首次匹配该 (label, value) 需经 `/dev/tty` 确认（记录 confirm.toml）；`$PICKTUI_PICK=off`/无 tty/无 label 跳过确认；无法唯一确定回退 TUI |

无 TTY 退化：无 query 取首个候选；有 query 过滤后取首个匹配，
**无匹配报错退出 1**（绝不静默回退首个，避免脚本拿到毫不相干的选中值）。
`--auto` 无唯一解时报错退出 1。候选为空报 `picktui pick: no candidates` 退出 1。

## menu

```
picktui menu <label> <cand...>
```

多值缩写运行时菜单：数字 1-9 直选（无输入时）、输入过滤、高亮、视口滚动、
选择记忆（key = label）。无 `/dev/tty` 时取首个候选输出 + stderr 警告，退出 0。
取消退出 130。参数不足（<label> + 至少 1 个候选）退出 2。

## filter（纯函数）

```
echo -e "build\tCompile\ndev\tServer" | picktui filter --json --query "de" [--mode substring|token|fuzzy] [--sep <chars>]
```

- stdin：结构化候选行（`key<TAB>description`，规则见总览）。
- stdout：过滤后的 JSON 数组，**保留输入顺序**：
  - `value`：选中值（key）
  - `desc`：描述（无则空串）
  - `ranges`：query 命中的 rune 索引区间 `[start,end)`（已排序合并），
    绑定层据此分段高亮；空查询恒为 `[]`（非 null）
- 空输入 → `[]`，退出 0（空输入不是错误）。
- 示例（substring 多关键字 + token 模式）：

```console
$ printf 'electron:dev\nelectron:build\nmydev\n' | picktui filter --json --query 'ele dev'
[{"value":"electron:dev","desc":"","ranges":[[0,3],[9,12]]}]

$ printf 'electron:dev\nelectron:build\n' | picktui filter --json --query 'ele dev' --mode token --sep ':_' | jq -c '.[].value'
"electron:dev"
```

## resolve（纯函数）

```
printf 'build\nrelease\nserve\n' | picktui resolve --json --query re
→ {"value":"release"}，退出 0
printf 'release\nrestart\n' | picktui resolve --json --query re
→ （stdout 空），退出 1
```

解析规则：精确匹配（大小写敏感）优先 → 否则大小写不敏感的唯一前缀匹配。
无匹配/多个前缀匹配/空 query → stdout 空 + 退出 1（调用方回退 TUI 或报错）。
`--mode/--sep` 接受但仅作一致性（解析规则与 filter 模式无关）。

## hist（文件状态）

```
picktui hist set "git p" pull     # 成功：stdout 空，退出 0；错误：stderr + 退出 1
picktui hist get "git p"          # {"label":"git p","last":"pull"}，退出 0
picktui hist get "never set"      # {"label":"never set","last":""}，退出 0
```

`last` 为空串即无记录（绑定层直接判 falsy）。用法错误退出 2。

## confirm（文件状态）

```
picktui confirm check "npm run" release   # {"confirmed":false} | {"confirmed":true}
picktui confirm add  "npm run" release    # 幂等；输出确认后的实际状态 {"confirmed":true}
picktui confirm add  "" x                 # label 为空不落盘 → {"confirmed":false}
```

错误（写盘失败）：stderr + 退出 1。

## 版本

`picktui version`（或 `-v/--version`）输出 `picktui <semver>`（如 `picktui 0.1.0`）。
绑定层发布时声明最低引擎版本，运行前用 version 校验，避免协议错配。

## 兼容性承诺

- 字段名/形状/退出码为协议 v1 的一部分，仅在 v2（大版本）破坏性变更；
  新增字段只增不删，旧绑定忽略新字段。
- 引擎改动统一进 picktui 仓库；宿主（shr 等）只保留薄包装与品牌对齐
  （SetName/SetOffEnv/SetDataDir/SetMapResolver），不复制渲染或匹配逻辑。