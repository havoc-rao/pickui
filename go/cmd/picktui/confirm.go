// confirm — 协议子命令：自动匹配首次确认（文件状态，纯数据面）。
//
//	picktui confirm check <label> <value>  → {"confirmed":true|false}
//	picktui confirm add <label> <value>    → 记录后输出 {"confirmed":<实际状态>}
//	                                        （幂等；label 为空时不落盘 → false）
//
// 存储文件：DataDir()/confirm.toml，与 --auto 命令的确认记录同一份。
package main

import (
	"fmt"
	"io"

	"github.com/havoc-rao/picktui/go"
)

// cmdConfirm 实现 `confirm check|add [...]`。
func cmdConfirm(stdout, stderr io.Writer, args []string) int {
	if len(args) < 3 {
		fmt.Fprintf(stderr, "usage: picktui confirm %s\n", confirmUsage(args))
		return 2
	}
	sub, label, value := args[0], args[1], args[2]
	switch sub {
	case "check":
		if err := encodeJSON(stdout, map[string]bool{"confirmed": picktui.IsConfirmed(label, value)}); err != nil {
			fmt.Fprintf(stderr, "picktui confirm: 编码 JSON 失败: %v\n", err)
			return 1
		}
		return 0
	case "add":
		if err := picktui.Confirm(label, value); err != nil {
			fmt.Fprintf(stderr, "picktui confirm: %v\n", err)
			return 1
		}
		if err := encodeJSON(stdout, map[string]bool{"confirmed": picktui.IsConfirmed(label, value)}); err != nil {
			fmt.Fprintf(stderr, "picktui confirm: 编码 JSON 失败: %v\n", err)
			return 1
		}
		return 0
	default:
		fmt.Fprintf(stderr, "usage: picktui confirm %s\n", confirmUsage(args))
		return 2
	}
}

func confirmUsage(args []string) string {
	if len(args) > 0 && args[0] == "add" && len(args) < 3 {
		return "add <label> <value>"
	}
	if len(args) > 0 && args[0] == "check" && len(args) < 3 {
		return "check <label> <value>"
	}
	return "check <label> <value> | add <label> <value>"
}
