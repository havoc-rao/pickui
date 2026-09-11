// Package picktui_test: 候选来源（命令/reader/--map 转换）黑盒测试。
package picktui_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/havoc-rao/picktui/go"
)

// ---- 取数 + 转换（--map） ----

func TestStructuredFromCommandMapped(t *testing.T) {
	// --from 取原始输出（冒号分隔），--map 转成 key<TAB>des
	cands, err := picktui.StructuredFromCommandMapped(
		`printf 'build:Compile the project\ndev:Start dev server\n'`,
		`tr ':' '\t'`,
	)
	if err != nil {
		t.Fatalf("StructuredFromCommandMapped: %v", err)
	}
	want := []picktui.Candidate{
		{Value: "build", Desc: "Compile the project"},
		{Value: "dev", Desc: "Start dev server"},
	}
	wantCands(t, cands, want)
}

func TestStructuredFromCommandMappedFromErr(t *testing.T) {
	// 取数命令失败 → 错误透传（含 stderr）
	_, err := picktui.StructuredFromCommandMapped(`exit 3`, `tr 'a-z' 'A-Z'`)
	if err == nil {
		t.Fatal("StructuredFromCommandMapped with failing source: want error, got nil")
	}
}

func TestStructuredFromCommandMappedMapperErr(t *testing.T) {
	// mapper 非零退出 → 错误透传（含 stderr）
	_, err := picktui.StructuredFromCommandMapped(`printf 'build:ok\n'`, `cat; exit 1`)
	if err == nil {
		t.Fatal("StructuredFromCommandMapped with failing mapper: want error, got nil")
	}
}

func TestStructuredFromReaderMapped(t *testing.T) {
	// --map 接 stdin：从 reader 读原始输入，mapper 输出结构化行
	r := strings.NewReader("a 1\nb 2\n")
	cands, err := picktui.StructuredFromReaderMapped(r, `awk '{print $1 "\t" $2}'`)
	if err != nil {
		t.Fatalf("StructuredFromReaderMapped: %v", err)
	}
	want := []picktui.Candidate{
		{Value: "a", Desc: "1"},
		{Value: "b", Desc: "2"},
	}
	wantCands(t, cands, want)
}

// ---- mapper 脚本文件预检 ----

func TestMapperScriptMissingAbs(t *testing.T) {
	// 绝对路径脚本不存在 → fail-fast 返回 MapperScriptError，而非执行 node 刷堆栈
	_, err := picktui.StructuredFromCommandMapped(`printf 'x\n'`, `node /definitely/not/here/runner.mjs`)
	var mse *picktui.MapperScriptError
	if !errors.As(err, &mse) {
		t.Fatalf("want MapperScriptError, got %v", err)
	}
	if mse.Path != "/definitely/not/here/runner.mjs" {
		t.Fatalf("missing path = %q, want /definitely/not/here/runner.mjs", mse.Path)
	}
}

func TestMapperScriptMissingBareName(t *testing.T) {
	// 纯文件名带脚本扩展名（cwd 不存在）→ 命中
	_, err := picktui.StructuredFromCommandMapped(`printf 'x\n'`, `python3 this_file_surely_does_not_exist_12345.py`)
	var mse *picktui.MapperScriptError
	if !errors.As(err, &mse) {
		t.Fatalf("want MapperScriptError for bare .py name, got %v", err)
	}
	if mse.Path != "this_file_surely_does_not_exist_12345.py" {
		t.Fatalf("path = %q", mse.Path)
	}
}

func TestMapperScriptTildeExpanded(t *testing.T) {
	// ~ 前缀展开后判断；缺失时 Path 为展开后的绝对路径
	_, err := picktui.StructuredFromCommandMapped(`printf 'x\n'`, `node ~/.picktui/nope/runner.mjs`)
	var mse *picktui.MapperScriptError
	if !errors.As(err, &mse) {
		t.Fatalf("want MapperScriptError for ~ path, got %v", err)
	}
	if mse.Path == "~/.picktui/nope/runner.mjs" || !strings.HasPrefix(mse.Path, "/") {
		t.Fatalf("path not tilde-expanded: %q", mse.Path)
	}
}

func TestMapperScriptPresentNoIntercept(t *testing.T) {
	// 无路径特征 token（tr / node -e 等）→ 不预检，正常执行
	cands, err := picktui.StructuredFromCommandMapped(`printf 'a:1\nb:2\n'`, `tr ':' '\t'`)
	if err != nil {
		t.Fatalf("tr mapper should not be intercepted: %v", err)
	}
	want := []picktui.Candidate{
		{Value: "a", Desc: "1"},
		{Value: "b", Desc: "2"},
	}
	wantCands(t, cands, want)

	_, err = picktui.StructuredFromCommandMapped(`printf 'x\n'`, `node -e 'console.log("1")'`)
	var mse *picktui.MapperScriptError
	if errors.As(err, &mse) {
		t.Fatalf("node -e should not trigger script check, got %v", mse)
	}
}

// ---- reader 来源 ----

func TestCandidatesFromReader(t *testing.T) {
	got := picktui.CandidatesFromReader(strings.NewReader("a 1\nb 2\n* main\n\n"))
	if len(got) != 3 || got[0] != "a 1" || got[1] != "b 2" || got[2] != "main" {
		t.Fatalf("CandidatesFromReader = %v, want [a 1 b 2 main]", got)
	}
}

func TestStructuredFromReaderEmpty(t *testing.T) {
	if got := picktui.StructuredFromReader(strings.NewReader("")); len(got) != 0 {
		t.Fatalf("StructuredFromReader(empty) = %v, want []", got)
	}
}

// ---- StringCandidates ----

func TestStringCandidates(t *testing.T) {
	got := picktui.StringCandidates([]string{"a", "b"})
	want := []picktui.Candidate{{Value: "a"}, {Value: "b"}}
	wantCands(t, got, want)
}
