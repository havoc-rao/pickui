// Package picktui_test: Pick 命令非交互路径黑盒测试
// （env off / openTTY 失败注入，均不触碰真实 TTY）。
package picktui_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/havoc-rao/picktui/go"
)

// pickOff 以 PICKTUI_PICK=off 运行 Pick（确定性非交互）。
func pickOff(t *testing.T, args ...string) (code int, out, errw string) {
	t.Helper()
	t.Setenv("PICKTUI_PICK", "off")
	var ob, eb bytes.Buffer
	code = picktui.Pick(args, &ob, &eb)
	return code, ob.String(), eb.String()
}

func TestPickNonInteractiveNoQuery(t *testing.T) {
	code, out, _ := pickOff(t, "alpha", "beta", "gamma")
	if code != 0 || out != "alpha\n" {
		t.Fatalf("Pick = %d, out %q, want 0, \"alpha\\n\"", code, out)
	}
}

func TestPickNonInteractiveQueryFiltersFirstMatch(t *testing.T) {
	code, out, _ := pickOff(t, "-q", "beta", "alpha", "beta", "gamma")
	if code != 0 || out != "beta\n" {
		t.Fatalf("Pick(-q beta) = %d, out %q, want 0, \"beta\\n\"", code, out)
	}
}

func TestPickNonInteractiveQueryNoMatch(t *testing.T) {
	code, _, errw := pickOff(t, "-q", "zzz", "alpha", "beta")
	if code != 1 {
		t.Fatalf("Pick(-q zzz) = %d, want 1", code)
	}
	if !strings.Contains(errw, `picktui pick: no match for "zzz"`) {
		t.Fatalf("stderr = %q, want no-match error", errw)
	}
}

func TestPickNoCandidates(t *testing.T) {
	// 无候选来源（无位置参数、stdin 为空/终端）→ 报错而非静默成功
	code, _, errw := pickOff(t)
	if code != 1 {
		t.Fatalf("Pick() = %d, want 1", code)
	}
	if !strings.Contains(errw, "picktui pick: no candidates") {
		t.Fatalf("stderr = %q, want no-candidates error", errw)
	}
}

func TestPickMapWithPositionalIsUsageError(t *testing.T) {
	code, _, errw := pickOff(t, "--map", "tr 'a-z' 'A-Z'", "pos1")
	if code != 2 {
		t.Fatalf("Pick(--map + positional) = %d, want 2", code)
	}
	if !strings.Contains(errw, "--map 只与 --from 或 stdin 配合") {
		t.Fatalf("stderr = %q, want usage error text", errw)
	}
}

func TestPickFromCommand(t *testing.T) {
	code, out, _ := pickOff(t, "--from", `printf 'main\ndevelop\n'`, "-q", "dev")
	if code != 0 || out != "develop\n" {
		t.Fatalf("Pick(--from) = %d, out %q, want 0, \"develop\\n\"", code, out)
	}
}

func TestPickFromCommandError(t *testing.T) {
	code, _, errw := pickOff(t, "--from", `echo boom >&2; exit 3`)
	if code != 1 {
		t.Fatalf("Pick(--from failing) = %d, want 1", code)
	}
	if !strings.Contains(errw, "boom") || !strings.Contains(errw, "exit status 3") {
		t.Fatalf("stderr = %q, want merged stderr diagnostics", errw)
	}
}

func TestPickFromMapped(t *testing.T) {
	code, out, errw := pickOff(t, "--from", `printf 'build:Compile\ndev:Server\n'`, "--map", `tr ':' '\t'`, "-q", "dev")
	if code != 0 || out != "dev\n" {
		t.Fatalf("Pick(--from --map) = %d, out %q, want 0, \"dev\\n\" (stderr: %s)", code, out, errw)
	}
}

func TestPickMapperScriptMissingRendersGuidance(t *testing.T) {
	code, _, errw := pickOff(t, "--from", `printf 'x\n'`, "--map", "node /definitely/not/here/runner.mjs")
	if code != 1 {
		t.Fatalf("Pick(missing mapper script) = %d, want 1", code)
	}
	for _, want := range []string{
		"picktui pick: printf 'x\\n' | node /definitely/not/here/runner.mjs:",
		"map 脚本文件不存在: /definitely/not/here/runner.mjs",
		"→ 脚本可能已被移动、删除或重命名",
		"`picktui add <cmd> --pick '<cmd>' --map '<新命令>'`",
	} {
		if !strings.Contains(errw, want) {
			t.Fatalf("stderr = %q, want containing %q", errw, want)
		}
	}
}

func TestPickTokenMode(t *testing.T) {
	code, out, _ := pickOff(t, "--sep", ":_", "-q", "ele dev", "electron:dev", "electron:build", "mydev")
	if code != 0 || out != "electron:dev\n" {
		t.Fatalf("Pick(--sep) = %d, out %q, want 0, electron:dev", code, out)
	}
}

func TestPickFuzzyMode(t *testing.T) {
	code, out, _ := pickOff(t, "--fuzzy", "-q", "mwh", "mmbiz_wx_hav", "main")
	if code != 0 || out != "mmbiz_wx_hav\n" {
		t.Fatalf("Pick(--fuzzy) = %d, out %q, want 0, mmbiz_wx_hav", code, out)
	}
}

func TestPickSelect1(t *testing.T) {
	code, out, _ := pickOff(t, "-1", "-q", "uniq", "uniq", "alpha", "beta")
	if code != 0 || out != "uniq\n" {
		t.Fatalf("Pick(-1) = %d, out %q, want 0, \"uniq\\n\"", code, out)
	}
	// 多个匹配时不自动选中，走非交互首候选（query 过滤后首个）
	code, out, _ = pickOff(t, "-1", "-q", "a", "alpha", "beta", "aleph")
	if code != 0 || out != "alpha\n" {
		t.Fatalf("Pick(-1 ambiguous) = %d, out %q, want 0, \"alpha\\n\"", code, out)
	}
}

func TestPickAutoResolveOffEnvSkipsConfirm(t *testing.T) {
	dir := picktuiConfigDir(t)
	code, out, _ := pickOff(t, "--auto", "-q", "re", "build", "release", "serve")
	if code != 0 || out != "release\n" {
		t.Fatalf("Pick(--auto re) = %d, out %q, want 0, \"release\\n\"", code, out)
	}
	// PICKTUI_PICK=off 跳过确认：confirm.toml 不生成
	if _, err := os.Stat(filepath.Join(dir, "confirm.toml")); !os.IsNotExist(err) {
		t.Fatalf("confirm.toml should not be created in off mode: %v", err)
	}
}

func TestPickAutoResolveExactWins(t *testing.T) {
	code, out, _ := pickOff(t, "--auto", "-q", "dev", "dev", "dev:watch", "release")
	if code != 0 || out != "dev\n" {
		t.Fatalf("Pick(--auto dev) = %d, out %q, want 0, \"dev\\n\"", code, out)
	}
}

func TestPickAutoNoUniqueMatch(t *testing.T) {
	code, _, errw := pickOff(t, "--auto", "-q", "re", "release", "restart")
	if code != 1 {
		t.Fatalf("Pick(--auto ambiguous) = %d, want 1", code)
	}
	if !strings.Contains(errw, `picktui pick: no unique match for "re"`) {
		t.Fatalf("stderr = %q, want no-unique-match error", errw)
	}
}

func TestPickLabelSavesHistory(t *testing.T) {
	picktuiConfigDir(t)
	code, out, _ := pickOff(t, "--label", "npm run", "-q", "dev", "build", "dev", "serve")
	if code != 0 || out != "dev\n" {
		t.Fatalf("Pick(--label) = %d, out %q, want 0, \"dev\\n\"", code, out)
	}
	raw := picktui.LoadPickHistory()
	if got := picktui.LastPick(raw, "npm run"); got != "dev" {
		t.Fatalf("LastPick(npm run) = %q, want dev", got)
	}
}

func TestPickSelect1SavesHistory(t *testing.T) {
	dir := picktuiConfigDir(t)
	code, out, _ := pickOff(t, "--label", "git p", "-1", "-q", "pull", "pull", "push")
	if code != 0 || out != "pull\n" {
		t.Fatalf("Pick(-1 --label) = %d, out %q, want 0, \"pull\\n\"", code, out)
	}
	if got := picktui.LastPick(picktui.LoadPickHistory(), "git p"); got != "pull" {
		t.Fatalf("LastPick(git p) = %q, want pull", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "history.toml")); err != nil {
		t.Fatalf("history.toml not created: %v", err)
	}
}

func TestParseConfirmAnswer(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"y", true},
		{"Y", true},
		{"yes", true},
		{"YES", true},
		{" Yes \n", true},
		{"", false},
		{"\n", false},
		{"n", false},
		{"N", false},
		{"no", false},
		{"No", false},
		{"maybe", false},
	}
	for _, c := range cases {
		if got := picktui.ParseConfirmAnswer(c.in); got != c.want {
			t.Errorf("ParseConfirmAnswer(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPickUsesHostNameAndOffEnv(t *testing.T) {
	// shr 嵌入形态：SetName("shr") / SetOffEnv("SHR_PICK") → 消息与开关逐项对齐
	picktui.SetName("shr")
	picktui.SetOffEnv("SHR_PICK")
	t.Cleanup(func() {
		picktui.SetName("picktui")
		picktui.SetOffEnv("PICKTUI_PICK")
	})
	t.Setenv("SHR_PICK", "off")

	var ob, eb bytes.Buffer
	code := picktui.Pick([]string{"-q", "zzz", "alpha"}, &ob, &eb)
	if code != 1 {
		t.Fatalf("Pick = %d, want 1", code)
	}
	if !strings.Contains(eb.String(), `shr pick: no match for "zzz"`) {
		t.Fatalf("stderr = %q, want shr-branded message", eb.String())
	}
}
