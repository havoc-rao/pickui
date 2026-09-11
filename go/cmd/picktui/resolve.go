// resolve — 协议子命令：自动解析唯一候选（纯函数数据面）。
//
//	picktui resolve --json [--query <q>] [--mode substring|token|fuzzy] [--sep <chars>]
//
// stdin 按结构化候选读取；--query 精确命中（大小写敏感）或唯一前缀命中时
// stdout 输出 {"value":"<选中值>"} 且退出码 0；无匹配/多个前缀匹配时 stdout
// 为空、退出码 1（调用方据此回退 TUI 或报错）。
//
// --mode/--sep 与 filter 相同但仅作协议一致性接受（解析规则与 filter 无关，
// 自动解析恒为"精确 → 唯一前缀"）。
package main

import (
	"fmt"
	"io"

	"github.com/havoc-rao/picktui/go"
)

// cmdResolve 实现 `resolve --json [...]`：stdin 候选 → 唯一解析 → JSON。
func cmdResolve(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	var query, mode, sep string
	seenJSON := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			seenJSON = true
		case a == "--query":
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case len(a) > len("--query=") && a[:len("--query=")] == "--query=":
			query = a[len("--query="):]
		case a == "--mode":
			if i+1 < len(args) {
				i++
				mode = args[i]
			}
		case len(a) > len("--mode=") && a[:len("--mode=")] == "--mode=":
			mode = a[len("--mode="):]
		case a == "--sep":
			if i+1 < len(args) {
				i++
				sep = args[i]
			}
		case len(a) > len("--sep=") && a[:len("--sep=")] == "--sep=":
			sep = a[len("--sep="):]
		default:
			fmt.Fprintf(stderr, "picktui resolve: unknown flag %q\n", a)
			return 2
		}
	}
	if !seenJSON {
		fmt.Fprintln(stderr, "picktui resolve: missing --json (protocol requires explicit --json)")
		return 2
	}
	if _, err := parseFilterOptions(mode, sep); err != nil {
		fmt.Fprintf(stderr, "picktui resolve: %v\n", err)
		return 2
	}

	cands := picktui.StructuredFromReader(stdin)
	v, ok := picktui.AutoResolve(cands, query)
	if !ok {
		return 1 // stdout 保持为空（"空 + 退出码 1" 协议）
	}
	if err := encodeJSON(stdout, map[string]string{"value": v}); err != nil {
		fmt.Fprintf(stderr, "picktui resolve: 编码 JSON 失败: %v\n", err)
		return 1
	}
	return 0
}
