// Package picktui_test: filter 匹配逻辑黑盒测试。
package picktui_test

import (
	"testing"

	"github.com/havoc-rao/picktui/go"
)

// ---- 子串 AND ----

func TestFilterSubstringMultiKeyword(t *testing.T) {
	cands := []string{"mmbiz_wx_api_2_hav_3", "mmbiz_wx_api_2_hav_4", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "hav 3", opts)
	if len(got) != 1 || got[0] != "mmbiz_wx_api_2_hav_3" {
		t.Fatalf("FilterCandidates(hav 3) = %v, want [mmbiz_wx_api_2_hav_3]", got)
	}
}

func TestFilterSubstringCaseInsensitive(t *testing.T) {
	cands := []string{"Feature/Branch", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "feat", opts)
	if len(got) != 1 || got[0] != "Feature/Branch" {
		t.Fatalf("FilterCandidates(feat) = %v, want [Feature/Branch]", got)
	}
}

func TestFilterSubstringNoMatch(t *testing.T) {
	cands := []string{"main", "develop"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "xyz", opts)
	if len(got) != 0 {
		t.Fatalf("FilterCandidates(xyz) = %v, want []", got)
	}
}

func TestFilterSubstringSingleKeyword(t *testing.T) {
	cands := []string{"main", "master", "develop"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "ma", opts)
	if len(got) != 2 {
		t.Fatalf("FilterCandidates(ma) = %v, want 2 items", got)
	}
}

// ---- token 前缀 ----

func TestFilterTokenPrefixBasic(t *testing.T) {
	cands := []string{"mmbiz_wx_api_2_hav_3", "mmbiz_wx_api_2_hav_4", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: "_"}

	got := picktui.FilterCandidates(cands, "hav 3", opts)
	if len(got) != 1 || got[0] != "mmbiz_wx_api_2_hav_3" {
		t.Fatalf("FilterCandidates(hav 3, token) = %v, want [mmbiz_wx_api_2_hav_3]", got)
	}
}

func TestFilterTokenPrefixPrecision(t *testing.T) {
	// av should NOT match hav — token prefix, not substring
	cands := []string{"mmbiz_wx_api_2_hav_3", "mmbiz_wx_api_2_hav_4"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: "_"}

	got := picktui.FilterCandidates(cands, "av", opts)
	if len(got) != 0 {
		t.Fatalf("FilterCandidates(av, token) = %v, want [] (av is not a token prefix)", got)
	}
}

func TestFilterTokenPrefixDefaultSep(t *testing.T) {
	// Empty sep defaults to "_"
	cands := []string{"foo_bar", "baz_qux"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix}

	got := picktui.FilterCandidates(cands, "foo", opts)
	if len(got) != 1 || got[0] != "foo_bar" {
		t.Fatalf("FilterCandidates(foo, default sep) = %v, want [foo_bar]", got)
	}
}

func TestFilterTokenPrefixCustomSep(t *testing.T) {
	cands := []string{"foo-bar", "baz-qux"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: "-"}

	got := picktui.FilterCandidates(cands, "foo", opts)
	if len(got) != 1 || got[0] != "foo-bar" {
		t.Fatalf("FilterCandidates(foo, sep=-) = %v, want [foo-bar]", got)
	}
}

// ---- 模糊子序列 ----

func TestFilterFuzzyBasic(t *testing.T) {
	cands := []string{"mmbiz_wx_hav", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchFuzzy}

	got := picktui.FilterCandidates(cands, "mwh", opts)
	if len(got) != 1 || got[0] != "mmbiz_wx_hav" {
		t.Fatalf("FilterCandidates(mwh, fuzzy) = %v, want [mmbiz_wx_hav]", got)
	}
}

func TestFilterFuzzyNoMatch(t *testing.T) {
	cands := []string{"abc"}
	opts := picktui.FilterOptions{Mode: picktui.MatchFuzzy}

	got := picktui.FilterCandidates(cands, "xyz", opts)
	if len(got) != 0 {
		t.Fatalf("FilterCandidates(xyz, fuzzy) = %v, want []", got)
	}
}

func TestFilterFuzzyMultiKeyword(t *testing.T) {
	cands := []string{"mmbiz_wx_hav_3", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchFuzzy}

	got := picktui.FilterCandidates(cands, "mw h3", opts)
	if len(got) != 1 || got[0] != "mmbiz_wx_hav_3" {
		t.Fatalf("FilterCandidates(mw h3, fuzzy) = %v, want [mmbiz_wx_hav_3]", got)
	}
}

// ---- 边界 ----

func TestFilterEmptyQuery(t *testing.T) {
	cands := []string{"a", "b", "c"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "", opts)
	if len(got) != 3 {
		t.Fatalf("FilterCandidates('') = %v, want all 3", got)
	}
}

func TestFilterEmptyCandidates(t *testing.T) {
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}
	got := picktui.FilterCandidates(nil, "test", opts)
	if len(got) != 0 {
		t.Fatalf("FilterCandidates(nil) = %v, want []", got)
	}
}

func TestSplitKeywordsEmpty(t *testing.T) {
	if kws := picktui.SplitKeywords(""); kws != nil {
		t.Fatalf("SplitKeywords('') = %v, want nil", kws)
	}
	if kws := picktui.SplitKeywords("   "); kws != nil {
		t.Fatalf("SplitKeywords('   ') = %v, want nil", kws)
	}
}

func TestSplitKeywordsLowercase(t *testing.T) {
	kws := picktui.SplitKeywords("HAV 3")
	if len(kws) != 2 || kws[0] != "hav" || kws[1] != "3" {
		t.Fatalf("SplitKeywords(HAV 3) = %v, want [hav 3]", kws)
	}
}

// ---- CleanCandidates ----

func TestCleanCandidatesGitBranch(t *testing.T) {
	lines := []string{"* main", "  feature/x", "  develop", ""}
	got := picktui.CleanCandidates(lines)
	want := []string{"main", "feature/x", "develop"}
	if len(got) != len(want) {
		t.Fatalf("CleanCandidates = %v, want %v", got, want)
	}
	for i, v := range want {
		if got[i] != v {
			t.Fatalf("CleanCandidates[%d] = %q, want %q", i, got[i], v)
		}
	}
}

func TestCleanCandidatesDedup(t *testing.T) {
	lines := []string{"main", "main", "develop", "develop"}
	got := picktui.CleanCandidates(lines)
	if len(got) != 2 {
		t.Fatalf("CleanCandidates(dedup) = %v, want 2 items", got)
	}
}

func TestCleanCandidatesDetachedHead(t *testing.T) {
	lines := []string{"+ (HEAD detached at abc123)", "main"}
	got := picktui.CleanCandidates(lines)
	if len(got) != 2 {
		t.Fatalf("CleanCandidates(detached) = %v, want 2 items", got)
	}
	if got[0] != "(HEAD detached at abc123)" {
		t.Fatalf("CleanCandidates[0] = %q, want (HEAD detached at abc123)", got[0])
	}
}

func TestCleanCandidatesSkipEmpty(t *testing.T) {
	lines := []string{"", "  ", "\t", "main"}
	got := picktui.CleanCandidates(lines)
	if len(got) != 1 || got[0] != "main" {
		t.Fatalf("CleanCandidates(empty) = %v, want [main]", got)
	}
}

// ---- HighlightRanges ----

func TestHighlightRangesSubstring(t *testing.T) {
	ranges := picktui.HighlightRanges("mmbiz_hav_3", "hav 3", picktui.FilterOptions{Mode: picktui.MatchSubstring})
	// "hav" at [6,9), "3" at [10,11)
	foundHav := false
	found3 := false
	for _, r := range ranges {
		if r[0] == 6 && r[1] == 9 {
			foundHav = true
		}
		if r[0] == 10 && r[1] == 11 {
			found3 = true
		}
	}
	if !foundHav {
		t.Fatalf("HighlightRanges missing 'hav' at [6,9): %v", ranges)
	}
	if !found3 {
		t.Fatalf("HighlightRanges missing '3' at [10,11): %v", ranges)
	}
}

func TestHighlightRangesEmptyQuery(t *testing.T) {
	ranges := picktui.HighlightRanges("test", "", picktui.FilterOptions{Mode: picktui.MatchSubstring})
	if ranges != nil {
		t.Fatalf("HighlightRanges('') = %v, want nil", ranges)
	}
}

func TestHighlightRangesTokenPrefix(t *testing.T) {
	ranges := picktui.HighlightRanges("mmbiz_hav_3", "hav", picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: "_"})
	// "hav" is token prefix at [6,9)
	found := false
	for _, r := range ranges {
		if r[0] == 6 && r[1] == 9 {
			found = true
		}
	}
	if !found {
		t.Fatalf("HighlightRanges(token) missing 'hav' at [6,9): %v", ranges)
	}
}

func TestHighlightRangesFuzzy(t *testing.T) {
	ranges := picktui.HighlightRanges("mwh", "mwh", picktui.FilterOptions{Mode: picktui.MatchFuzzy})
	// All 3 characters highlighted, merged into [0,3)
	if len(ranges) != 1 {
		t.Fatalf("HighlightRanges(fuzzy) = %v, want 1 merged range", ranges)
	}
	if ranges[0] != [2]int{0, 3} {
		t.Fatalf("HighlightRanges(fuzzy) = %v, want [0,3)", ranges)
	}
}

func TestHighlightRangesMerged(t *testing.T) {
	// Adjacent ranges should merge: "ab" at [0,2) + "c" at [2,3) → [0,3)
	ranges := picktui.HighlightRanges("abc", "ab c", picktui.FilterOptions{Mode: picktui.MatchSubstring})
	if len(ranges) != 1 {
		t.Fatalf("HighlightRanges(merged) = %v, want 1 range", ranges)
	}
	if ranges[0] != [2]int{0, 3} {
		t.Fatalf("HighlightRanges(merged) = %v, want [0,3)", ranges)
	}
}

// ---- 结构化候选（key<TAB>des） ----

func wantCands(t *testing.T, got []picktui.Candidate, want []picktui.Candidate) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("got[%d] = %+v, want %+v", i, got[i], w)
		}
	}
}

func TestStructuredCandidatesTabSplit(t *testing.T) {
	lines := []string{
		"build\tCompile the project",
		"dev\tStart dev server",
		"test\tRun tests",
	}
	got := picktui.StructuredCandidates(lines)
	want := []picktui.Candidate{
		{Value: "build", Desc: "Compile the project"},
		{Value: "dev", Desc: "Start dev server"},
		{Value: "test", Desc: "Run tests"},
	}
	wantCands(t, got, want)
}

func TestStructuredCandidatesNoTab(t *testing.T) {
	// 无 TAB 行整行为 key，描述为空；清洗规则与 CleanCandidates 一致
	got := picktui.StructuredCandidates([]string{"main", "  develop  "})
	want := []picktui.Candidate{{Value: "main"}, {Value: "develop"}}
	wantCands(t, got, want)
}

func TestStructuredCandidatesGitBranchPrefix(t *testing.T) {
	// git branch 的 "* "/"+ " 标记清理照常，TAB 描述保留
	lines := []string{"* main\t(HEAD)", "+ (HEAD detached at abc123)\tdetached"}
	got := picktui.StructuredCandidates(lines)
	want := []picktui.Candidate{
		{Value: "main", Desc: "(HEAD)"},
		{Value: "(HEAD detached at abc123)", Desc: "detached"},
	}
	wantCands(t, got, want)
}

func TestStructuredCandidatesMergeDesc(t *testing.T) {
	// 先到 key 无描述，后到同名 key 补描述
	lines := []string{"build", "build\tCompile the project"}
	got := picktui.StructuredCandidates(lines)
	want := []picktui.Candidate{{Value: "build", Desc: "Compile the project"}}
	wantCands(t, got, want)
}

func TestStructuredCandidatesDedupKeepsFirstDesc(t *testing.T) {
	// 重复 key 去重保序，描述保留首个
	lines := []string{"build\tA", "build\tB"}
	got := picktui.StructuredCandidates(lines)
	want := []picktui.Candidate{{Value: "build", Desc: "A"}}
	wantCands(t, got, want)
}

func TestStructuredCandidatesSkipEmptyKey(t *testing.T) {
	// key 部分为空的行跳过（如 "\torphan desc"）
	got := picktui.StructuredCandidates([]string{"\torphan desc", "dev\tok"})
	want := []picktui.Candidate{{Value: "dev", Desc: "ok"}}
	wantCands(t, got, want)
}

func TestCleanCandidatesCompatibleWithTab(t *testing.T) {
	// 旧 API 在 TAB 行下只取 key，描述被剥离
	got := picktui.CleanCandidates([]string{"build\tCompile", "dev"})
	if len(got) != 2 || got[0] != "build" || got[1] != "dev" {
		t.Fatalf("CleanCandidates(tab) = %v, want [build dev]", got)
	}
}

func TestFilterStructuredCandidatesValueOnly(t *testing.T) {
	cands := []picktui.Candidate{
		{Value: "build", Desc: "Compile the project"},
		{Value: "dev", Desc: "Start dev server"},
	}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	// 关键字只命中描述 → 不匹配（过滤针对 key）
	got := picktui.FilterStructuredCandidates(cands, "compile", opts)
	if len(got) != 0 {
		t.Fatalf("FilterStructuredCandidates(compile) = %v, want [] (desc not matched)", got)
	}

	// 命中 key → 匹配，描述随候选保留
	got = picktui.FilterStructuredCandidates(cands, "dev", opts)
	want := []picktui.Candidate{{Value: "dev", Desc: "Start dev server"}}
	wantCands(t, got, want)
}

func TestFilterStructuredCandidatesEmptyQuery(t *testing.T) {
	cands := []picktui.Candidate{{Value: "a"}, {Value: "b"}}
	got := picktui.FilterStructuredCandidates(cands, "", picktui.FilterOptions{Mode: picktui.MatchSubstring})
	wantCands(t, got, cands)
}

// ---- AutoResolve（--auto 精确/唯一前缀自动选中） ----

func TestAutoResolveExactWins(t *testing.T) {
	// dev 既是精确候选又是 dev:watch 的前缀 → 精确优先
	cands := []picktui.Candidate{{Value: "dev"}, {Value: "dev:watch"}, {Value: "release"}}
	v, ok := picktui.AutoResolve(cands, "dev")
	if !ok || v != "dev" {
		t.Fatalf("AutoResolve(dev) = %q,%v want dev,true", v, ok)
	}
}

func TestAutoResolveUniquePrefix(t *testing.T) {
	cands := []picktui.Candidate{{Value: "build"}, {Value: "release"}, {Value: "serve"}}
	v, ok := picktui.AutoResolve(cands, "re")
	if !ok || v != "release" {
		t.Fatalf("AutoResolve(re) = %q,%v want release,true", v, ok)
	}
}

func TestAutoResolveMultiplePrefixes(t *testing.T) {
	cands := []picktui.Candidate{{Value: "release"}, {Value: "restart"}}
	if _, ok := picktui.AutoResolve(cands, "re"); ok {
		t.Fatal("AutoResolve(re) should fail with multiple prefix matches")
	}
}

func TestAutoResolveNoMatch(t *testing.T) {
	cands := []picktui.Candidate{{Value: "build"}, {Value: "dev"}}
	if _, ok := picktui.AutoResolve(cands, "xyz"); ok {
		t.Fatal("AutoResolve(xyz) should fail with no match")
	}
}

func TestAutoResolveCaseInsensitivePrefix(t *testing.T) {
	cands := []picktui.Candidate{{Value: "Release"}, {Value: "Build"}}
	v, ok := picktui.AutoResolve(cands, "REL")
	if !ok || v != "Release" {
		t.Fatalf("AutoResolve(REL) = %q,%v want Release,true", v, ok)
	}
}

func TestAutoResolveExactCaseSensitiveThenPrefix(t *testing.T) {
	// 精确匹配大小写敏感：DEV 不精确命中 dev，走唯一前缀（不敏感）回退
	cands := []picktui.Candidate{{Value: "dev"}}
	v, ok := picktui.AutoResolve(cands, "DEV")
	if !ok || v != "dev" {
		t.Fatalf("AutoResolve(DEV) = %q,%v want dev,true", v, ok)
	}
}

func TestAutoResolveEmptyQuery(t *testing.T) {
	if _, ok := picktui.AutoResolve([]picktui.Candidate{{Value: "dev"}}, ""); ok {
		t.Fatal("AutoResolve(empty) should fail")
	}
}

func TestAutoResolveDescriptionsIgnored(t *testing.T) {
	// 解析只看 Value（选中值），描述不参与
	cands := []picktui.Candidate{{Value: "release", Desc: "build for prod"}, {Value: "serve", Desc: "dev server"}}
	v, ok := picktui.AutoResolve(cands, "rel")
	if !ok || v != "release" {
		t.Fatalf("AutoResolve(rel) = %q,%v want release,true", v, ok)
	}
}

// ---- 多关键字空格过滤（子串 AND 跨冒号分隔） ----

func TestFilterSubstringSpaceKeywordsColonScript(t *testing.T) {
	// 用户场景：输入 "ele dev" 匹配 electron:dev（子串 AND，空格拆词）
	cands := []string{"electron:dev", "electron:build", "serve", "dev"}
	opts := picktui.FilterOptions{Mode: picktui.MatchSubstring}

	got := picktui.FilterCandidates(cands, "ele dev", opts)
	if len(got) != 1 || got[0] != "electron:dev" {
		t.Fatalf("FilterCandidates(ele dev) = %v, want [electron:dev]", got)
	}
}

// ---- token 前缀多分隔符集合 ----

func TestFilterTokenPrefixMultiSepColon(t *testing.T) {
	cands := []string{"electron:dev", "electron:build", "mydev"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: ":_"}

	got := picktui.FilterCandidates(cands, "ele dev", opts)
	// ele → token "electron" 前缀；dev → token "dev" 前缀；mydev 的 token 不以 dev 开头
	if len(got) != 1 || got[0] != "electron:dev" {
		t.Fatalf("FilterCandidates(ele dev, sep=:_) = %v, want [electron:dev]", got)
	}
}

func TestFilterTokenPrefixMultiSepMixed(t *testing.T) {
	cands := []string{"release_npm:build", "release", "npm:build"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: ":_"}

	got := picktui.FilterCandidates(cands, "npm build", opts)
	// npm 命中 token npm（冒号后），build 命中 token build（冒号后）
	if len(got) != 2 {
		t.Fatalf("FilterCandidates(npm build, sep=:_) = %v, want 2 items", got)
	}
}

func TestFilterTokenPrefixSingleSepBackwardCompat(t *testing.T) {
	// 单字符分隔符行为与原来一致（av 不误匹配 hav）
	cands := []string{"mmbiz_wx_api_2_hav_3", "main"}
	opts := picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: "_"}

	if got := picktui.FilterCandidates(cands, "av", opts); len(got) != 0 {
		t.Fatalf("FilterCandidates(av) = %v, want [] (no token starts with av)", got)
	}
	if got := picktui.FilterCandidates(cands, "hav 3", opts); len(got) != 1 {
		t.Fatalf("FilterCandidates(hav 3) = %v, want 1 item", got)
	}
}

func TestHighlightRangesTokenPrefixMultiSep(t *testing.T) {
	ranges := picktui.HighlightRanges("electron:dev", "ele dev",
		picktui.FilterOptions{Mode: picktui.MatchTokenPrefix, Sep: ":_"})
	// "ele" at [0,3), "dev" at [9,12)
	foundEle, foundDev := false, false
	for _, r := range ranges {
		if r[0] == 0 && r[1] == 3 {
			foundEle = true
		}
		if r[0] == 9 && r[1] == 12 {
			foundDev = true
		}
	}
	if !foundEle {
		t.Fatalf("HighlightRanges missing 'ele' at [0,3): %v", ranges)
	}
	if !foundDev {
		t.Fatalf("HighlightRanges missing 'dev' at [9,12): %v", ranges)
	}
}
