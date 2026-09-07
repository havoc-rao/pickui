// Command pickui: 候选选择器引擎的独立二进制。
//
// 提供协议子命令供跨语言绑定（TS/JS）与 shell 脚本消费，同时是 shr 等宿主
// 嵌入 Pick/Menu 之外的独立部署形态：
//
//	pickui pick [flags] [cand...]     交互过滤选择（stdout 选中值；无 TTY 自动退化）
//	pickui _menu <label> <cand...>    TUI 菜单（数字 1-9 直选；无 TTY 取首个 + 警告）
//	pickui filter --json [...]        纯函数：stdin 候选 → JSON 过滤结果
//	pickui resolve --json [...]       纯函数：stdin 候选 → 唯一解析结果
//	pickui hist get|set <label> [...] 选择记忆（文件状态）
//	pickui confirm check|add [...]    自动匹配首次确认（文件状态）
//
// 退出码协议：0 成功 / 1 运行错误（含 resolve 无唯一解）/ 2 用法错误 / 130 取消。
// 协议细节见 docs/protocol.md。
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/havoc-rao/pickui/go"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run 分发协议子命令（流可注入，便于测试）。
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return 2
	}
	switch args[0] {
	case "pick":
		return pickui.Pick(args[1:], stdout, stderr)
	case "menu", "_menu":
		return pickui.Menu(args[1:], stdout, stderr)
	case "filter":
		return cmdFilter(stdin, stdout, stderr, args[1:])
	case "resolve":
		return cmdResolve(stdin, stdout, stderr, args[1:])
	case "hist":
		return cmdHist(stdout, stderr, args[1:])
	case "confirm":
		return cmdConfirm(stdout, stderr, args[1:])
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "pickui %s\n", pickui.Version)
		return 0
	default:
		fmt.Fprintf(stderr, "pickui: unknown command %q (see: pickui help)\n", args[0])
		return 2
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `pickui — TUI candidate picker engine & protocol

Usage:
  pickui pick [flags] [cand...]              filter-pick dynamic candidates (TUI; no TTY falls back)
  pickui _menu <label> <cand...>             TUI menu for multi-value abbreviations
  pickui filter --json [...]                 filter stdin candidates, JSON out (pure)
  pickui resolve --json [...]                auto-resolve unique match, JSON out (pure)
  pickui hist get <label> | set <label> <value>   selection memory (file state)
  pickui confirm check <label> <value> | add <label> <value>  first-auto-match confirmations (file state)
  pickui help | version

Exit codes: 0 ok / 1 runtime error / 2 usage error / 130 cancelled.
Protocol details: docs/protocol.md
`)
}
