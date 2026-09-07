// Package pickui 的 filter 部分：候选匹配引擎——关键字过滤 → 高亮 → 自动解析。
//
// 纯函数无副作用，不依赖 TUI，可独立单测；TUI 与协议子命令（filter/resolve）
// 共用同一份实现，保证任何宿主拿到的过滤与高亮效果一致。
//
// 三种匹配模式：
//   - 子串 AND（默认）：空格分关键字，全部命中才保留，大小写不敏感
//   - token 前缀（--sep <chars>）：按分隔符集合切 token（默认 _，可多字符如 ":_"），
//     每关键字匹配某 token 前缀
//   - 子序列模糊（--fuzzy）：关键字为子序列
package pickui

import (
	"sort"
	"strings"
)

// MatchMode 匹配模式。
type MatchMode int

const (
	MatchSubstring   MatchMode = iota // 子串 AND（默认）
	MatchTokenPrefix                  // token 前缀（--sep）
	MatchFuzzy                        // 子序列模糊（--fuzzy）
)

// FilterOptions 过滤选项。
type FilterOptions struct {
	Mode MatchMode
	Sep  string // 仅 MatchTokenPrefix 用，分隔符集合（默认 "_"，可传多字符如 ":_"）
}

// SplitKeywords 将查询字符串拆分为小写关键字列表。
// 空格分词、小写化、过滤空串。空查询返回 nil。
func SplitKeywords(query string) []string {
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return nil
	}
	kws := make([]string, len(fields))
	for i, f := range fields {
		kws[i] = strings.ToLower(f)
	}
	return kws
}

// FilterCandidates 返回匹配全部关键字的候选列表，保留原始顺序。
// 空查询返回全部候选。
func FilterCandidates(cands []string, query string, opts FilterOptions) []string {
	kws := SplitKeywords(query)
	if len(kws) == 0 {
		return cands
	}
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		if matchCandidate(c, kws, opts) {
			out = append(out, c)
		}
	}
	return out
}

// FilterStructuredCandidates 返回匹配全部关键字的候选（按 Value 匹配），保留原始顺序。
// 描述不参与过滤——过滤永远针对"选中什么"而非"展示什么"。空查询返回全部候选。
func FilterStructuredCandidates(cands []Candidate, query string, opts FilterOptions) []Candidate {
	kws := SplitKeywords(query)
	if len(kws) == 0 {
		return cands
	}
	out := make([]Candidate, 0, len(cands))
	for _, c := range cands {
		if matchCandidate(c.Value, kws, opts) {
			out = append(out, c)
		}
	}
	return out
}

// AutoResolve 尝试把查询字符串解析为唯一候选（供 --auto 自动选中）：
//
//  1. 精确匹配（大小写敏感）直接命中——用户打的词与候选完全一致时优先，即使它同时是别的候选的前缀；
//  2. 否则做大小写不敏感的唯一前缀匹配——如 npm run 候选里 re 唯一前缀命中 release；
//
// 无匹配或多个前缀匹配时返回 ok=false，由调用方回退 TUI（query 预填）或报错。
func AutoResolve(cands []Candidate, query string) (value string, ok bool) {
	if query == "" {
		return "", false
	}
	for _, c := range cands {
		if c.Value == query {
			return c.Value, true
		}
	}
	lq := strings.ToLower(query)
	only := ""
	for _, c := range cands {
		if strings.HasPrefix(strings.ToLower(c.Value), lq) {
			if only != "" && only != c.Value {
				return "", false // 多个前缀匹配，无法唯一确定
			}
			only = c.Value
		}
	}
	if only != "" {
		return only, true
	}
	return "", false
}

func matchCandidate(s string, kws []string, opts FilterOptions) bool {
	switch opts.Mode {
	case MatchTokenPrefix:
		return matchTokenPrefixAll(s, kws, opts.Sep)
	case MatchFuzzy:
		return matchFuzzyAll(s, kws)
	default:
		return matchSubstringAll(s, kws)
	}
}

// matchSubstringAll 子串 AND：所有关键字都需作为子串出现（大小写不敏感）。
func matchSubstringAll(s string, kws []string) bool {
	lower := strings.ToLower(s)
	for _, kw := range kws {
		if !strings.Contains(lower, kw) {
			return false
		}
	}
	return true
}

// matchTokenPrefixAll token 前缀：按分隔符集合切 token，每关键字须匹配某 token 的前缀。
// 关键优势：av 不会误匹配 hav（前缀而非子串），比子串精准。
// sep 支持多字符（如 ":_"）表示"任一分隔符都切 token"，适配冒号分隔的 npm 脚本名等。
func matchTokenPrefixAll(s string, kws []string, sep string) bool {
	if sep == "" {
		sep = "_"
	}
	lower := strings.ToLower(s)
	tokens := splitTokens(lower, sep)
	for _, kw := range kws {
		matched := false
		for _, tok := range tokens {
			if strings.HasPrefix(tok, kw) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

// splitTokens 按分隔符集合切分 s：seps 中每个 rune 都是分隔符，连续/首尾分隔符
// 不产生空 token。seps 为空时整串作为单个 token（此时匹配退化为整串前缀判断）。
func splitTokens(s, seps string) []string {
	if seps == "" {
		return []string{s}
	}
	sepSet := make(map[rune]bool, len(seps))
	for _, r := range seps {
		sepSet[r] = true
	}
	var tokens []string
	var cur strings.Builder
	for _, r := range s {
		if sepSet[r] {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// matchFuzzyAll 子序列模糊：所有关键字都需作为子序列出现（大小写不敏感）。
func matchFuzzyAll(s string, kws []string) bool {
	lower := strings.ToLower(s)
	for _, kw := range kws {
		if !isSubsequence(kw, lower) {
			return false
		}
	}
	return true
}

// isSubsequence 判断 pat 是否为 s 的子序列（不要求连续）。
func isSubsequence(pat, s string) bool {
	if len(pat) == 0 {
		return true
	}
	pi := 0
	for si := 0; si < len(s) && pi < len(pat); si++ {
		if s[si] == pat[pi] {
			pi++
		}
	}
	return pi == len(pat)
}

// HighlightRanges 返回 s 中匹配关键字的 [start, end) rune 索引区间列表（已排序合并）。
// 供 TUI 分段染色渲染与 filter --json 协议的 ranges 字段使用。空查询返回 nil。
func HighlightRanges(s string, query string, opts FilterOptions) [][2]int {
	kws := SplitKeywords(query)
	if len(kws) == 0 {
		return nil
	}
	lowerRunes := []rune(strings.ToLower(s))
	var ranges [][2]int

	switch opts.Mode {
	case MatchTokenPrefix:
		sep := opts.Sep
		if sep == "" {
			sep = "_"
		}
		sepRunes := []rune(sep)
		tokRanges := splitRuneRanges(lowerRunes, sepRunes)
		for _, kw := range kws {
			kwRunes := []rune(kw)
			if len(kwRunes) == 0 {
				continue
			}
			for _, tr := range tokRanges {
				tok := lowerRunes[tr[0]:tr[1]]
				if hasRunePrefix(tok, kwRunes) {
					ranges = append(ranges, [2]int{tr[0], tr[0] + len(kwRunes)})
					break
				}
			}
		}
	case MatchFuzzy:
		for _, kw := range kws {
			positions := subsequenceRunePositions([]rune(kw), lowerRunes)
			for _, p := range positions {
				ranges = append(ranges, [2]int{p, p + 1})
			}
		}
	default:
		for _, kw := range kws {
			kwRunes := []rune(kw)
			if len(kwRunes) == 0 {
				continue
			}
			idx := runeIndex(lowerRunes, kwRunes)
			if idx >= 0 {
				ranges = append(ranges, [2]int{idx, idx + len(kwRunes)})
			}
		}
	}

	return mergeRanges(ranges)
}

// ---- rune 辅助函数 ----

// runeIndex 在 s 中查找 substr 的首个出现位置（rune 索引），未找到返回 -1。
func runeIndex(s, substr []rune) int {
	if len(substr) == 0 || len(substr) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if s[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// hasRunePrefix 判断 s 是否以 prefix 开头（rune 切片）。
func hasRunePrefix(s, prefix []rune) bool {
	if len(prefix) > len(s) {
		return false
	}
	for i, r := range prefix {
		if s[i] != r {
			return false
		}
	}
	return true
}

// splitRuneRanges 按分隔符集合切分 runes，返回每个非空 token 的 [start, end) rune 索引区间。
// seps 中每个 rune 都是一个分隔符（如 ":_"）；seps 为空时整串视为一个 token。
func splitRuneRanges(runes, seps []rune) [][2]int {
	if len(seps) == 0 {
		if len(runes) > 0 {
			return [][2]int{{0, len(runes)}}
		}
		return nil
	}
	sepSet := make(map[rune]bool, len(seps))
	for _, r := range seps {
		sepSet[r] = true
	}
	var ranges [][2]int
	start := 0
	for i, r := range runes {
		if sepSet[r] {
			if i > start {
				ranges = append(ranges, [2]int{start, i})
			}
			start = i + 1
		}
	}
	if start < len(runes) {
		ranges = append(ranges, [2]int{start, len(runes)})
	}
	return ranges
}

// subsequenceRunePositions 返回 pat 作为 s 子序列匹配时的各字符 rune 位置。
// 不完整匹配返回 nil。
func subsequenceRunePositions(pat, s []rune) []int {
	if len(pat) == 0 {
		return nil
	}
	var positions []int
	pi := 0
	for si := 0; si < len(s) && pi < len(pat); si++ {
		if s[si] == pat[pi] {
			positions = append(positions, si)
			pi++
		}
	}
	if pi < len(pat) {
		return nil
	}
	return positions
}

// mergeRanges 排序并合并重叠的 [start, end) 区间。
func mergeRanges(ranges [][2]int) [][2]int {
	if len(ranges) == 0 {
		return nil
	}
	sort.Slice(ranges, func(i, j int) bool {
		if ranges[i][0] != ranges[j][0] {
			return ranges[i][0] < ranges[j][0]
		}
		return ranges[i][1] < ranges[j][1]
	})
	merged := [][2]int{ranges[0]}
	for i := 1; i < len(ranges); i++ {
		last := &merged[len(merged)-1]
		if ranges[i][0] <= last[1] {
			if ranges[i][1] > last[1] {
				last[1] = ranges[i][1]
			}
		} else {
			merged = append(merged, ranges[i])
		}
	}
	return merged
}
