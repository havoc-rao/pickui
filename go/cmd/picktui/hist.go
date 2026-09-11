// hist — 协议子命令：选择记忆（文件状态，纯数据面）。
//
//	picktui hist get <label>            → {"label":"git p","last":"pull"}（无记录 last 为空串）
//	picktui hist set <label> <value>    → 成功 stdout 为空；错误信息走 stderr + 退出码 1
//
// 存储文件：DataDir()/history.toml（默认 $PICKTUI_CONFIG_DIR →
// $XDG_CONFIG_HOME/picktui → ~/.config/picktui），与 TUI 选择记忆同一份。
package main

import (
	"fmt"
	"io"

	"github.com/havoc-rao/picktui/go"
)

// cmdHist 实现 `hist get|set [...]`。
func cmdHist(stdout, stderr io.Writer, args []string) int {
	if len(args) < 2 {
		fmt.Fprintf(stderr, "usage: picktui hist %s\n", histUsage(args))
		return 2
	}
	sub := args[0]
	switch sub {
	case "get":
		label := args[1]
		last := picktui.LastPick(picktui.LoadPickHistory(), label)
		if err := encodeJSON(stdout, map[string]string{"label": label, "last": last}); err != nil {
			fmt.Fprintf(stderr, "picktui hist: 编码 JSON 失败: %v\n", err)
			return 1
		}
		return 0
	case "set":
		if len(args) < 3 {
			fmt.Fprintln(stderr, "usage: picktui hist set <label> <value>")
			return 2
		}
		if err := picktui.SavePick(args[1], args[2]); err != nil {
			fmt.Fprintf(stderr, "picktui hist: %v\n", err)
			return 1
		}
		return 0 // stdout 为空（协议：成功无输出）
	default:
		fmt.Fprintf(stderr, "usage: picktui hist %s\n", histUsage(args))
		return 2
	}
}

func histUsage(args []string) string {
	if len(args) > 0 && (args[0] == "set" || args[0] == "get" && len(args) < 2) {
		return fmt.Sprintf("%s <label> [<value>]", args[0])
	}
	return "get <label> | set <label> <value>"
}
