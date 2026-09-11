// filter — 协议子命令：纯函数数据面。
//
//	picktui filter --json [--query <q>] [--mode substring|token|fuzzy] [--sep <chars>]
//
// stdin 按结构化候选读取（key<TAB>description，规则见 picktui.Candidate），
// stdout 输出 JSON 数组：
//
//	[{"value":"electron:dev","desc":"dev server","ranges":[[0,3],[9,12]]}, ...]
//
// ranges 为命中的 rune 索引 [start,end) 区间（空查询恒为 []），供绑定层做
// 分段高亮。--mode 默认 substring；--sep 仅 token 模式（默认 "_"）。
// 输出恒为 JSON；`--json` 显式声明（绑定层可校验）。
package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/havoc-rao/picktui/go"
)

// filterResult 是 filter 协议输出的单条结果。
type filterResult struct {
	Value  string   `json:"value"`
	Desc   string   `json:"desc"`
	Ranges [][2]int `json:"ranges"`
}

// cmdFilter 实现 `filter --json [...]`：stdin 候选 → 过滤 → JSON。
// 退出码：0 成功（含空输入输出 []）；1 运行错误；2 用法错误。
func cmdFilter(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
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
			fmt.Fprintf(stderr, "picktui filter: unknown flag %q\n", a)
			return 2
		}
	}
	if !seenJSON {
		fmt.Fprintln(stderr, "picktui filter: missing --json (protocol requires explicit --json)")
		return 2
	}
	opts, err := parseFilterOptions(mode, sep)
	if err != nil {
		fmt.Fprintf(stderr, "picktui filter: %v\n", err)
		return 2
	}

	cands := picktui.StructuredFromReader(stdin)
	hits := picktui.FilterStructuredCandidates(cands, query, opts)
	out := make([]filterResult, 0, len(hits))
	for _, c := range hits {
		ranges := picktui.HighlightRanges(c.Value, query, opts)
		if ranges == nil {
			ranges = [][2]int{}
		}
		out = append(out, filterResult{Value: c.Value, Desc: c.Desc, Ranges: ranges})
	}
	if err := encodeJSON(stdout, out); err != nil {
		fmt.Fprintf(stderr, "picktui filter: 编码 JSON 失败: %v\n", err)
		return 1
	}
	return 0
}

// resolve — 协议子命令的 filter 模式解析辅助，供 cmdFilter 与 cmdResolve 共用。
func parseFilterOptions(mode, sep string) (picktui.FilterOptions, error) {
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}
	switch mode {
	case "", "substring":
		// 默认
	case "token":
		opts.Mode = picktui.MatchTokenPrefix
		opts.Sep = sep
	case "fuzzy":
		opts.Mode = picktui.MatchFuzzy
	default:
		return opts, fmt.Errorf("unknown --mode %q (substring|token|fuzzy)", mode)
	}
	return opts, nil
}

// encodeJSON 输出 JSON 数组/对象（不转义 HTML，末尾换行）。
func encodeJSON(w io.Writer, v interface{}) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
