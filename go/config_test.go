// Package pickui_test: 泛化点（SetName/SetOffEnv/SetDataDir/DataDir 解析）测试。
package pickui_test

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/havoc-rao/pickui/go"
)

// resetGlobals 还原包级配置默认值（config_test 与其它测试共享进程）。
func resetGlobals(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		pickui.SetName("pickui")
		pickui.SetOffEnv("PICKUI_PICK")
		pickui.SetDataDir("")
	})
}

func TestDefaults(t *testing.T) {
	resetGlobals(t)
	if got := pickui.Name(); got != "pickui" {
		t.Fatalf("default Name() = %q, want pickui", got)
	}
	if got := pickui.OffEnv(); got != "PICKUI_PICK" {
		t.Fatalf("default OffEnv() = %q, want PICKUI_PICK", got)
	}
}

func TestSetNameOffEnv(t *testing.T) {
	resetGlobals(t)
	pickui.SetName("shr")
	pickui.SetOffEnv("SHR_PICK")
	if got := pickui.Name(); got != "shr" {
		t.Fatalf("Name() = %q, want shr", got)
	}
	if got := pickui.OffEnv(); got != "SHR_PICK" {
		t.Fatalf("OffEnv() = %q, want SHR_PICK", got)
	}
}

func TestDataDirEnvPrecedence(t *testing.T) {
	resetGlobals(t)
	envDir := filepath.Join(t.TempDir(), "from-pickui-env")
	xdgDir := filepath.Join(t.TempDir(), "from-xdg")
	t.Setenv("PICKUI_CONFIG_DIR", envDir)
	t.Setenv("XDG_CONFIG_HOME", xdgDir)
	if got := pickui.DataDir(); got != envDir {
		t.Fatalf("DataDir() = %q, want %q (PICKUI_CONFIG_DIR wins)", got, envDir)
	}
}

func TestDataDirXDG(t *testing.T) {
	resetGlobals(t)
	t.Setenv("PICKUI_CONFIG_DIR", "")
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	if got, want := pickui.DataDir(), filepath.Join(base, "pickui"); got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestDataDirHomeFallback(t *testing.T) {
	resetGlobals(t)
	t.Setenv("PICKUI_CONFIG_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := pickui.DataDir(), filepath.Join(home, ".config", "pickui"); got != want {
		t.Fatalf("DataDir() = %q, want %q", got, want)
	}
}

func TestSetDataDirExplicitWins(t *testing.T) {
	resetGlobals(t)
	t.Setenv("PICKUI_CONFIG_DIR", t.TempDir())
	pickui.SetDataDir("/custom/pickui-state")
	if got := pickui.DataDir(); got != "/custom/pickui-state" {
		t.Fatalf("DataDir() = %q, want explicit set dir", got)
	}
}

func TestSetMapResolver(t *testing.T) {
	resetGlobals(t)
	t.Setenv("PICKUI_PICK", "off") // 非交互：不碰 TTY

	// 无钩子：裸名按字面执行（找不到命令 → 运行错误，非用法错误）
	{
		var out, errw bytes.Buffer
		code := pickui.Pick([]string{"--from", "printf 'a\\nb\\n'", "--map", "this_cmd_does_not_exist_xyz"}, &out, &errw)
		if code != 1 {
			t.Fatalf("Pick with unresolvable bare mapper = %d, want 1", code)
		}
		if !strings.Contains(errw.String(), "this_cmd_does_not_exist_xyz") {
			t.Fatalf("stderr should mention mapper: %q", errw.String())
		}
	}

	// 钩子：裸名 → 实际命令（shr 嵌入形态：'npm-run' → 完整命令）
	pickui.SetMapResolver(func(m string) string {
		if m == "versionify" {
			return "tr ':' '\\t'"
		}
		return m
	})
	var out, errw bytes.Buffer
	code := pickui.Pick([]string{"--from", `printf 'build:Compile\ndev:Server\n'`, "--map", "versionify", "-q", "dev"}, &out, &errw)
	if code != 0 {
		t.Fatalf("Pick with mapped resolver = %d, want 0 (stderr: %s)", code, errw.String())
	}
	if got := out.String(); got != "dev\n" {
		t.Fatalf("selected = %q, want dev", got)
	}
}
