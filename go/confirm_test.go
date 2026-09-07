// Package pickui_test: 自动匹配首次确认记录（confirm.toml）测试。
package pickui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/havoc-rao/pickui/go"
)

func TestConfirmStateRoundTrip(t *testing.T) {
	dir := pickuiConfigDir(t)
	if pickui.IsConfirmed("npm run", "release") {
		t.Fatal("fresh state should not be confirmed")
	}
	if err := pickui.Confirm("npm run", "release"); err != nil {
		t.Fatal(err)
	}
	if !pickui.IsConfirmed("npm run", "release") {
		t.Fatal("confirmed value should be visible after save")
	}
	// 其它 label 或值不受影响
	if pickui.IsConfirmed("npm run", "dev") {
		t.Fatal("unconfirmed value leaked")
	}
	if pickui.IsConfirmed("docker exec", "release") {
		t.Fatal("confirmation leaked across labels")
	}
	// 文件生成在配置目录
	if _, err := os.Stat(filepath.Join(dir, "confirm.toml")); err != nil {
		t.Fatalf("confirm.toml not created: %v", err)
	}
}

func TestConfirmIdempotent(t *testing.T) {
	pickuiConfigDir(t)
	for i := 0; i < 3; i++ {
		if err := pickui.Confirm("git co", "main"); err != nil {
			t.Fatal(err)
		}
	}
	// 同一 (label, value) 只记录一次
	if got := len(pickui.LoadConfirmed()["git co"]); got != 1 {
		t.Fatalf("confirmed values = %d, want 1", got)
	}
}

func TestConfirmMultipleValuesPerLabel(t *testing.T) {
	pickuiConfigDir(t)
	if err := pickui.Confirm("npm run", "dev"); err != nil {
		t.Fatal(err)
	}
	if err := pickui.Confirm("npm run", "build"); err != nil {
		t.Fatal(err)
	}
	if err := pickui.Confirm("docker exec", "web"); err != nil {
		t.Fatal(err)
	}
	m := pickui.LoadConfirmed()
	if len(m["npm run"]) != 2 {
		t.Fatalf("npm run values = %v, want 2", m["npm run"])
	}
	if len(m["docker exec"]) != 1 {
		t.Fatalf("docker exec values = %v, want 1", m["docker exec"])
	}
}

func TestConfirmEmptyLabelNoop(t *testing.T) {
	dir := pickuiConfigDir(t)
	if err := pickui.Confirm("", "x"); err != nil {
		t.Fatal(err)
	}
	// 无 label 不落盘（无状态可记）
	if _, err := os.Stat(filepath.Join(dir, "confirm.toml")); !os.IsNotExist(err) {
		t.Fatalf("confirm.toml should not exist for empty label: %v", err)
	}
}

func TestConfirmCorruptedFileTolerated(t *testing.T) {
	dir := pickuiConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "confirm.toml"), []byte("not toml [[["), 0o644); err != nil {
		t.Fatal(err)
	}
	// 损坏文件 → 视为无记录，不 panic、不误报
	if pickui.IsConfirmed("npm run", "release") {
		t.Fatal("corrupted file should yield no confirmations")
	}
	if got := len(pickui.LoadConfirmed()); got != 0 {
		t.Fatalf("LoadConfirmed on corrupted file = %v, want empty", got)
	}
	// 覆盖写回后恢复可用
	if err := pickui.Confirm("npm run", "dev"); err != nil {
		t.Fatal(err)
	}
	if !pickui.IsConfirmed("npm run", "dev") {
		t.Fatal("confirm after corrupted file should work")
	}
}
