# picktui go — 引擎包（Go）

模块：`github.com/havoc-rao/picktui/go`，包名 `picktui`。引擎本体，三种消费方式：

1. **嵌入**：`import picktui "github.com/havoc-rao/picktui/go"`，函数直调（同进程，无进程开销）
2. **二进制**：`go build ./cmd/picktui` → 协议子命令（见 ../docs/integration/protocol.md）
3. **宿主对齐**：`SetName / SetOffEnv / SetDataDir / SetMapResolver`（shr 即以此接入）

## 嵌入 API（按需取用）

| 需求 | API |
|---|---|
| 只要过滤/高亮 | `FilterStructuredCandidates` / `FilterCandidates` / `HighlightRanges` |
| 只要 --auto 解析 | `AutoResolve` |
| 只要交互选择器 | `Run(opts, tty)` / `Options` / `Result`（调用方打开 /dev/tty） |
| 只要选择记忆 | `SavePick / LastPick / LoadPickHistory` |
| 只要首次确认 | `Confirm / IsConfirmed / ParseConfirmAnswer` |
| 完整 CLI 命令嵌入 | `Pick(args, out, errw) int` / `Menu(args, out, errw) int` |
| 宿主对齐 | `SetName / SetOffEnv / SetDataDir / SetMapResolver` |

约定：`Pick/Menu` 的选中值（数据）直写 `out` 不带色码；错误/警告写 `errw`，
行首前缀在 TTY 下着红粗体，管道/重定向自动降级纯文本——`$(...)` 安全。
退出码：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消。
非交互（`$PICKTUI_PICK=off` 或无 /dev/tty）：无 query 取首个，有 query 过滤取首。

## 开发

```console
$ go test ./... -count=1
```

测试：filter 匹配矩阵、TUI 按键行为（model 级）、history 迁移、confirm、
DataDir 解析、Pick/Menu 非交互路径、协议子命令 JSON 往返（cmd/picktui）。