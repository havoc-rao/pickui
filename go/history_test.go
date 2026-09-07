// Package pickui_test: 选择记忆（history.toml）加载/保存与旧版 picks.toml 自动迁移测试。
package pickui_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/havoc-rao/pickui/go"
)

// pickuiConfigDir 返回隔离的数据目录（PICKUI_CONFIG_DIR=临时目录），并自动还原。
func pickuiConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("PICKUI_CONFIG_DIR", dir)
	return dir
}

func TestHistorySaveLoad(t *testing.T) {
	dir := pickuiConfigDir(t)
	if err := pickui.SavePick("git p", "pull"); err != nil {
		t.Fatal(err)
	}
	raw := pickui.LoadPickHistory()
	if got := pickui.LastPick(raw, "git p"); got != "pull" {
		t.Fatalf("LastPick(git p) = %q, want pull", got)
	}
	// 记忆写在新文件 history.toml，而非旧名 picks.toml
	if _, err := os.Stat(filepath.Join(dir, "history.toml")); err != nil {
		t.Fatalf("history.toml not created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "picks.toml")); !os.IsNotExist(err) {
		t.Fatalf("picks.toml should not exist: %v", err)
	}
}

func TestHistoryMigratesLegacyPicks(t *testing.T) {
	dir := pickuiConfigDir(t)
	legacy := "# pickui pick history — last selection per key (auto-managed)\n" +
		"[\"git p\"]\n" +
		"last = \"pull\"\n" +
		"\n" +
		"[\"git co\"]\n" +
		"last = \"main\"\n"
	if err := os.WriteFile(filepath.Join(dir, "picks.toml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	raw := pickui.LoadPickHistory()
	if got := pickui.LastPick(raw, "git p"); got != "pull" {
		t.Fatalf("migrated LastPick(git p) = %q, want pull", got)
	}
	if got := pickui.LastPick(raw, "git co"); got != "main" {
		t.Fatalf("migrated LastPick(git co) = %q, want main", got)
	}
	// 迁移后：history.toml 生成、picks.toml 删除
	if _, err := os.Stat(filepath.Join(dir, "history.toml")); err != nil {
		t.Fatalf("history.toml not created after migration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "picks.toml")); !os.IsNotExist(err) {
		t.Fatalf("picks.toml should be removed after migration: %v", err)
	}
	// 二次加载直接走新文件，幂等
	if got := pickui.LastPick(pickui.LoadPickHistory(), "git p"); got != "pull" {
		t.Fatalf("second load LastPick(git p) = %q, want pull", got)
	}
}

func TestHistorySaveMigratesLegacyPicks(t *testing.T) {
	dir := pickuiConfigDir(t)
	legacy := "[\"git co\"]\nlast = \"main\"\n"
	if err := os.WriteFile(filepath.Join(dir, "picks.toml"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	// SavePick 内部先 LoadPickHistory（触发迁移），再写回新文件
	if err := pickui.SavePick("git p", "push"); err != nil {
		t.Fatal(err)
	}
	raw := pickui.LoadPickHistory()
	if got := pickui.LastPick(raw, "git co"); got != "main" {
		t.Fatalf("old entry lost after save: LastPick(git co) = %q, want main", got)
	}
	if got := pickui.LastPick(raw, "git p"); got != "push" {
		t.Fatalf("LastPick(git p) = %q, want push", got)
	}
	if _, err := os.Stat(filepath.Join(dir, "picks.toml")); !os.IsNotExist(err) {
		t.Fatalf("picks.toml should be removed after save: %v", err)
	}
}

func TestHistoryEmptyWhenMissing(t *testing.T) {
	pickuiConfigDir(t)
	if raw := pickui.LoadPickHistory(); len(raw) != 0 {
		t.Fatalf("expected empty history, got %v", raw)
	}
}

func TestHistoryCorruptedFileTolerated(t *testing.T) {
	dir := pickuiConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "history.toml"), []byte("not toml [[["), 0o644); err != nil {
		t.Fatal(err)
	}
	raw, err := pickui.LoadPickHistoryWithErr()
	if err == nil {
		t.Fatal("corrupted history.toml should surface an error from LoadPickHistoryWithErr")
	}
	if len(raw) != 0 {
		t.Fatalf("corrupted history should load as empty, got %v", raw)
	}
	// LoadPickHistory 静默容忍；后续写入覆盖恢复
	if got := pickui.LastPick(pickui.LoadPickHistory(), "git p"); got != "" {
		t.Fatalf("corrupted history LastPick = %q, want empty", got)
	}
	if err := pickui.SavePick("git p", "pull"); err != nil {
		t.Fatalf("save after corrupted file: %v", err)
	}
	if got := pickui.LastPick(pickui.LoadPickHistory(), "git p"); got != "pull" {
		t.Fatalf("LastPick after rewrite = %q, want pull", got)
	}
}
