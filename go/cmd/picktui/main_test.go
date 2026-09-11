// Package main: 协议子命令 JSON 往返测试（filter/resolve/hist/confirm，
// 全部无 TTY 依赖，直接注入流）。
package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/havoc-rao/picktui/go"
)

// runProto 以给定 stdin/args 执行协议子命令，返回退出码与输出。
func runProto(t *testing.T, stdin string, args ...string) (code int, out, errw string) {
	t.Helper()
	var ob, eb bytes.Buffer
	code = run(args, strings.NewReader(stdin), &ob, &eb)
	return code, ob.String(), eb.String()
}

// protoConfigDir 返回隔离数据目录（hist/confirm 协议用）。
func protoConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PICKTUI_CONFIG_DIR", dir)
	return dir
}

// ---- filter：JSON 往返 ----

func TestFilterJSONSubstring(t *testing.T) {
	code, out, errw := runProto(t, "build\tCompile the project\ndev\tStart dev server\nrelease\n",
		"filter", "--json", "--query", "dev")
	if code != 0 {
		t.Fatalf("filter = %d, want 0 (stderr: %s)", code, errw)
	}
	var got []struct {
		Value  string   `json:"value"`
		Desc   string   `json:"desc"`
		Ranges [][2]int `json:"ranges"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("filter output not JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].Value != "dev" || got[0].Desc != "Start dev server" {
		t.Fatalf("got %+v, want single dev hit with desc", got)
	}
	if len(got[0].Ranges) != 1 || got[0].Ranges[0] != [2]int{0, 3} {
		t.Fatalf("ranges = %v, want [[0,3]]", got[0].Ranges)
	}
}

func TestFilterJSONEmptyQueryAllCandidates(t *testing.T) {
	code, out, _ := runProto(t, "a\nb\nc\tC\n", "filter", "--json")
	if code != 0 {
		t.Fatalf("filter = %d, want 0", code)
	}
	var got []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("filter output not JSON: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d candidates, want 3", len(got))
	}
	// 空查询 → ranges 恒为 []（非 null，绑定层可直接消费）
	for i, c := range got {
		if c["ranges"] == nil {
			t.Fatalf("candidate %d ranges is null, want []", i)
		}
		if s, _ := c["ranges"].([]interface{}); len(s) != 0 {
			t.Fatalf("candidate %d ranges = %v, want empty", i, s)
		}
	}
}

func TestFilterJSONTokenMode(t *testing.T) {
	code, out, _ := runProto(t, "electron:dev\nelectron:build\nmydev\n",
		"filter", "--json", "--query", "ele dev", "--mode", "token", "--sep", ":_")
	if code != 0 {
		t.Fatalf("filter(token) = %d, want 0", code)
	}
	var got []struct {
		Value  string   `json:"value"`
		Ranges [][2]int `json:"ranges"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("filter output not JSON: %v", err)
	}
	if len(got) != 1 || got[0].Value != "electron:dev" {
		t.Fatalf("got %+v, want [electron:dev]", got)
	}
	// ele → [0,3)，dev → [9,12)
	want := [][2]int{{0, 3}, {9, 12}}
	if len(got[0].Ranges) != 2 || got[0].Ranges[0] != want[0] || got[0].Ranges[1] != want[1] {
		t.Fatalf("ranges = %v, want %v", got[0].Ranges, want)
	}
}

func TestFilterJSONFuzzyMode(t *testing.T) {
	code, out, _ := runProto(t, "mmbiz_wx_hav\nmain\n", "filter", "--json", "--query", "mwh", "--mode", "fuzzy")
	if code != 0 {
		t.Fatalf("filter(fuzzy) = %d, want 0", code)
	}
	if !strings.Contains(out, `"value":"mmbiz_wx_hav"`) || strings.Contains(out, `"value":"main"`) {
		t.Fatalf("filter(fuzzy) = %s, want only mmbiz_wx_hav", out)
	}
}

func TestFilterJSONEmptyInput(t *testing.T) {
	code, out, errw := runProto(t, "", "filter", "--json")
	if code != 0 {
		t.Fatalf("filter(empty) = %d, want 0 (stderr: %s)", code, errw)
	}
	if strings.TrimSpace(out) != "[]" {
		t.Fatalf("filter(empty) = %q, want []", out)
	}
}

func TestFilterJSONUsageErrors(t *testing.T) {
	if code, _, _ := runProto(t, "", "filter"); code != 2 {
		t.Fatalf("filter without --json = %d, want 2", code)
	}
	code, _, errw := runProto(t, "", "filter", "--json", "--mode", "bogus")
	if code != 2 {
		t.Fatalf("filter(bogus mode) = %d, want 2", code)
	}
	if !strings.Contains(errw, "unknown --mode") {
		t.Fatalf("stderr = %q, want unknown --mode error", errw)
	}
	code, _, _ = runProto(t, "", "filter", "--json", "--nope")
	if code != 2 {
		t.Fatalf("filter(unknown flag) = %d, want 2", code)
	}
}

// ---- resolve：唯一解析 ----

func TestResolveJSONExact(t *testing.T) {
	code, out, _ := runProto(t, "dev\ndev:watch\nrelease\n", "resolve", "--json", "--query", "dev")
	if code != 0 {
		t.Fatalf("resolve(exact) = %d, want 0", code)
	}
	if strings.TrimSpace(out) != `{"value":"dev"}` {
		t.Fatalf("resolve(exact) = %q, want {\"value\":\"dev\"}", out)
	}
}

func TestResolveJSONUniquePrefix(t *testing.T) {
	code, out, _ := runProto(t, "build\nrelease\nserve\n", "resolve", "--json", "--query", "re")
	if code != 0 || strings.TrimSpace(out) != `{"value":"release"}` {
		t.Fatalf("resolve(prefix) = %d, %q, want 0, {\"value\":\"release\"}", code, out)
	}
}

func TestResolveJSONNoUniqueMatch(t *testing.T) {
	// 多个前缀匹配 / 无匹配 / 空 query → stdout 空 + 退出码 1
	for _, q := range []string{"re", "xyz", ""} {
		code, out, _ := runProto(t, "release\nrestart\n", "resolve", "--json", "--query", q)
		if code != 1 {
			t.Fatalf("resolve(%q) = %d, want 1", q, code)
		}
		if out != "" {
			t.Fatalf("resolve(%q) stdout = %q, want empty", q, out)
		}
	}
}

func TestResolveJSONModeAccepted(t *testing.T) {
	// --mode/--sep 协议一致性接受，不影响解析结果
	code, out, _ := runProto(t, "mmbiz_wx_hav\nmain\n", "resolve", "--json", "--query", "mwh", "--mode", "fuzzy")
	// 解析规则与 filter 模式无关：mwh 不是任何候选的前缀 → 无唯一解（退出 1）
	if code != 1 || out != "" {
		t.Fatalf("resolve(fuzzy) = %d, %q, want prefix-resolution semantics (1, empty)", code, out)
	}
	code, out, _ = runProto(t, "mmbiz_wx_hav\nmain\n", "resolve", "--json", "--query", "mmbiz", "--mode", "fuzzy")
	if code != 0 || strings.TrimSpace(out) != `{"value":"mmbiz_wx_hav"}` {
		t.Fatalf("resolve(fuzzy prefix) = %d, %q, want {\"value\":\"mmbiz_wx_hav\"}", code, out)
	}
}

// ---- hist：选择记忆 ----

func TestHistGetSetRoundTrip(t *testing.T) {
	protoConfigDir(t)
	code, out, errw := runProto(t, "", "hist", "set", "git p", "pull")
	if code != 0 || out != "" {
		t.Fatalf("hist set = %d, out %q, want 0 and empty stdout (stderr: %s)", code, out, errw)
	}
	code, out, _ = runProto(t, "", "hist", "get", "git p")
	if code != 0 {
		t.Fatalf("hist get = %d, want 0", code)
	}
	if strings.TrimSpace(out) != `{"label":"git p","last":"pull"}` {
		t.Fatalf("hist get = %q, want {\"label\":\"git p\",\"last\":\"pull\"}", out)
	}
	// 无记录 → last 空串（绑定层可直接判空）
	code, out, _ = runProto(t, "", "hist", "get", "never set")
	if code != 0 || !strings.Contains(out, `"last":""`) {
		t.Fatalf("hist get(missing) = %d, %q, want last:\"\"", code, out)
	}
	// JSON 可解码
	var dec map[string]string
	if err := json.Unmarshal([]byte(out), &dec); err != nil {
		t.Fatalf("hist get output not JSON: %v", err)
	}
}

func TestHistUsageErrors(t *testing.T) {
	if code, _, _ := runProto(t, "", "hist"); code != 2 {
		t.Fatalf("hist without subcommand = %d, want 2", code)
	}
	if code, _, _ := runProto(t, "", "hist", "get"); code != 2 {
		t.Fatalf("hist get without label = %d, want 2", code)
	}
	if code, _, _ := runProto(t, "", "hist", "set", "label-only"); code != 2 {
		t.Fatalf("hist set without value = %d, want 2", code)
	}
	if code, _, _ := runProto(t, "", "hist", "bogus", "x"); code != 2 {
		t.Fatalf("hist bogus subcommand = %d, want 2", code)
	}
}

func TestHistSetErrorPath(t *testing.T) {
	// 数据目录被普通文件占用 → MkdirAll 失败 → 退出码 1 + stderr 消息
	f := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PICKTUI_CONFIG_DIR", f)
	code, out, errw := runProto(t, "", "hist", "set", "git p", "pull")
	if code != 1 || out != "" {
		t.Fatalf("hist set(blocked dir) = %d, out %q, want 1 and empty stdout", code, out)
	}
	if !strings.Contains(errw, "picktui hist:") {
		t.Fatalf("stderr = %q, want picktui hist: error", errw)
	}
}

// ---- confirm：首次确认记录 ----

func TestConfirmCheckAddRoundTrip(t *testing.T) {
	protoConfigDir(t)
	code, out, _ := runProto(t, "", "confirm", "check", "npm run", "release")
	if code != 0 || strings.TrimSpace(out) != `{"confirmed":false}` {
		t.Fatalf("confirm check(fresh) = %d, %q, want {\"confirmed\":false}", code, out)
	}
	code, out, errw := runProto(t, "", "confirm", "add", "npm run", "release")
	if code != 0 || strings.TrimSpace(out) != `{"confirmed":true}` {
		t.Fatalf("confirm add = %d, %q, want {\"confirmed\":true} (stderr: %s)", code, out, errw)
	}
	// 幂等：重复 add 不新增记录
	if code, _, _ := runProto(t, "", "confirm", "add", "npm run", "release"); code != 0 {
		t.Fatalf("confirm add(idempotent) = %d, want 0", code)
	}
	m := picktui.LoadConfirmed()
	if len(m["npm run"]) != 1 {
		t.Fatalf("confirmed values = %d, want 1", len(m["npm run"]))
	}
	// 其它 value / label 不受影响
	code, out, _ = runProto(t, "", "confirm", "check", "npm run", "dev")
	if code != 0 || strings.TrimSpace(out) != `{"confirmed":false}` {
		t.Fatalf("confirm check(dev) = %d, %q, want false", code, out)
	}
}

func TestConfirmEmptyLabelNoop(t *testing.T) {
	dir := protoConfigDir(t)
	code, out, _ := runProto(t, "", "confirm", "add", "", "x")
	if code != 0 || strings.TrimSpace(out) != `{"confirmed":false}` {
		t.Fatalf("confirm add(empty label) = %d, %q, want {\"confirmed\":false}", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "confirm.toml")); !os.IsNotExist(err) {
		t.Fatalf("confirm.toml should not exist for empty label: %v", err)
	}
}

func TestConfirmUsageErrors(t *testing.T) {
	if code, _, _ := runProto(t, "", "confirm"); code != 2 {
		t.Fatalf("confirm without subcommand = %d, want 2", code)
	}
	if code, _, _ := runProto(t, "", "confirm", "check", "only-label"); code != 2 {
		t.Fatalf("confirm check without value = %d, want 2", code)
	}
	if code, _, _ := runProto(t, "", "confirm", "bogus", "a", "b"); code != 2 {
		t.Fatalf("confirm bogus subcommand = %d, want 2", code)
	}
}

// ---- 顶层分发 ----

func TestRunVersionAndHelp(t *testing.T) {
	code, out, _ := runProto(t, "", "version")
	if code != 0 || !strings.Contains(out, "picktui 0.1.0") {
		t.Fatalf("version = %d, %q, want picktui 0.1.0", code, out)
	}
	code, out, _ = runProto(t, "", "help")
	if code != 0 {
		t.Fatalf("help = %d, want 0", code)
	}
	for _, want := range []string{"picktui pick", "filter", "resolve", "hist", "confirm", "Exit codes"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q", want)
		}
	}
}

func TestRunPickMenuDispatch(t *testing.T) {
	// pick：非交互（off）直接输出选中值
	t.Setenv("PICKTUI_PICK", "off")
	code, out, errw := runProto(t, "", "pick", "alpha", "beta")
	if code != 0 || out != "alpha\n" {
		t.Fatalf("pick = %d, %q, want 0, alpha (stderr: %s)", code, out, errw)
	}
	// menu：缺参数 → 用法错误 2（TUI 交互路径依赖真实 /dev/tty，由库测试覆盖）
	code, out, _ = runProto(t, "", "menu", "only-label")
	if code != 2 {
		t.Fatalf("menu(usage) = %d, want 2", code)
	}
	if out != "" {
		t.Fatalf("menu(usage) stdout = %q, want empty", out)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	code, _, errw := runProto(t, "", "bogus")
	if code != 2 {
		t.Fatalf("unknown command = %d, want 2", code)
	}
	if !strings.Contains(errw, "unknown command") {
		t.Fatalf("stderr = %q, want unknown command error", errw)
	}
}

func TestRunNoArgs(t *testing.T) {
	code, _, errw := runProto(t, "")
	if code != 2 {
		t.Fatalf("no args = %d, want 2", code)
	}
	if !strings.Contains(errw, "Usage:") {
		t.Fatalf("stderr = %q, want usage", errw)
	}
}

// ---- 数据目录来自动态 DataDir（与库一致） ----

func TestHistSharesLibraryDataDir(t *testing.T) {
	dir := protoConfigDir(t)
	// 库 API 写入 → 协议子命令读同一文件
	if err := picktui.SavePick("shared", "value"); err != nil {
		t.Fatal(err)
	}
	code, out, _ := runProto(t, "", "hist", "get", "shared")
	if code != 0 || !strings.Contains(out, `"last":"value"`) {
		t.Fatalf("hist get = %d, %q, want last:value from library share", code, out)
	}
	if _, err := os.Stat(filepath.Join(dir, "history.toml")); err != nil {
		t.Fatalf("history.toml not created: %v", err)
	}
}
