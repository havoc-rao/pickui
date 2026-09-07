// Package pickui: Pick/Menu 非交互退化的内部测试
// （openTTY 注入失败，不触碰真实 /dev/tty）。
package pickui

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// noTTY 返回一个永远失败的 tty 打开器（注入非交互退化）。
func noTTY() (*os.File, error) { return nil, errors.New("no /dev/tty in test") }

// pickNoTTY 注入 tty 打开失败运行 pickWith。
func pickNoTTY(t *testing.T, args ...string) (code int, out, errw string) {
	t.Helper()
	var ob, eb bytes.Buffer
	code = pickWith(args, &ob, &eb, noTTY)
	return code, ob.String(), eb.String()
}

func TestPickOpenTTYErrorFallsBackNonInteractive(t *testing.T) {
	// 未设 off 但打不开 /dev/tty → 同样走非交互（首候选）
	code, out, errw := pickNoTTY(t, "alpha", "beta")
	if code != 0 || out != "alpha\n" {
		t.Fatalf("Pick(no tty) = %d, out %q, want 0, \"alpha\\n\" (stderr: %s)", code, out, errw)
	}
}

func TestMenuUsageError(t *testing.T) {
	var ob, eb bytes.Buffer
	code := menuWith([]string{"only-label"}, &ob, &eb, noTTY)
	if code != 2 {
		t.Fatalf("Menu(<1 arg) = %d, want 2", code)
	}
	if got := eb.String(); got != "usage: pickui _menu <label> <cand...>\n" {
		t.Fatalf("stderr = %q, want usage text", got)
	}
	if ob.String() != "" {
		t.Fatalf("stdout = %q, want empty", ob.String())
	}
}

func TestMenuNonInteractiveFallback(t *testing.T) {
	var ob, eb bytes.Buffer
	code := menuWith([]string{"db", "mysql", "postgres"}, &ob, &eb, noTTY)
	if code != 0 {
		t.Fatalf("Menu(no tty) = %d, want 0", code)
	}
	if ob.String() != "mysql\n" {
		t.Fatalf("stdout = %q, want first candidate mysql", ob.String())
	}
	if !strings.Contains(eb.String(), "多值缩写 \"db\" 在非交互环境下取首个候选 \"mysql\"") {
		t.Fatalf("stderr = %q, want fallback warning", eb.String())
	}
}

func TestMenuUsageTextFollowsName(t *testing.T) {
	SetName("shr")
	defer SetName("pickui")
	if got := MenuUsage(); got != "usage: shr _menu <label> <cand...>" {
		t.Fatalf("MenuUsage() = %q, want shr-branded usage", got)
	}
}
