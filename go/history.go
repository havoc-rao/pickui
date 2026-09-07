// history — 选择记忆：记住每个选择器/歧义点上次选中的候选，
// 下次弹窗时光标默认停在该候选上（--label 即记忆键）。
//
// 文件格式（TOML，位于数据目录 history.toml）：
//
//	# pickui history — last selection per key (auto-managed)
//	["git p"]
//	last = "pull"
//
// 纯文件存储、无 UI，不依赖 TUI，可独立单测。
package pickui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// HistoryFileName 选择记忆文件名，位于数据目录（DataDir()）。
// 曾用名 picks.toml 因与 `pick` 命令（动态候选过滤选择器）混淆而弃用。
const HistoryFileName = "history.toml"

// LegacyPickHistoryFileName 旧版选择记忆文件名，仅用于首次读取时自动迁移。
const LegacyPickHistoryFileName = "picks.toml"

// PickHistoryPath 返回选择记忆文件路径。
func PickHistoryPath() string {
	return filepath.Join(DataDir(), HistoryFileName)
}

// LoadPickHistory 解码 history.toml 为原始 map（label → {last: value}）。
// 新位置不存在但旧版 picks.toml 存在时自动迁移（读取后写入新位置并删除旧文件）；
// 文件不存在或损坏时返回空 map（静默忽略错误，记忆只是锦上添花）。
func LoadPickHistory() map[string]interface{} {
	raw, _ := LoadPickHistoryWithErr()
	if raw == nil {
		return map[string]interface{}{}
	}
	return raw
}

// LoadPickHistoryWithErr 与 LoadPickHistory 相同，但返回解析错误供调用方诊断。
// 文件不存在时返回 (空map, nil)；文件损坏时返回 (空map, error)。
func LoadPickHistoryWithErr() (map[string]interface{}, error) {
	raw := map[string]interface{}{}
	if _, err := os.Stat(PickHistoryPath()); err == nil {
		if _, err := toml.DecodeFile(PickHistoryPath(), &raw); err != nil {
			return map[string]interface{}{}, fmt.Errorf("解析 %s 失败: %w", PickHistoryPath(), err)
		}
		return raw, nil
	}
	legacy := filepath.Join(DataDir(), LegacyPickHistoryFileName)
	if _, err := os.Stat(legacy); err == nil {
		if _, err := toml.DecodeFile(legacy, &raw); err != nil {
			return map[string]interface{}{}, fmt.Errorf("解析旧版 %s 失败: %w", legacy, err)
		}
		if err := writeHistory(raw); err != nil {
			// 迁移写入失败不阻断读取——旧数据仍在内存中可用
			fmt.Fprintf(os.Stderr, "%s: 迁移选择记忆到 %s 失败: %v\n", Name(), PickHistoryPath(), err)
		} else {
			_ = os.Remove(legacy)
		}
		return raw, nil
	}
	return map[string]interface{}{}, nil
}

// LastPick 从记忆中取出 label 上次选中的值；无记录返回 ""。
func LastPick(raw map[string]interface{}, label string) string {
	if tbl, ok := raw[label].(map[string]interface{}); ok {
		if last, ok := tbl["last"].(string); ok {
			return last
		}
	}
	return ""
}

// SavePick 记录 label 的上次选中值，保留其它 label 的记忆后写回。
// 返回写入错误（如磁盘满、权限不足），调用方应记录但不必中断主流程——
// 选择记忆只是锦上添花的功能。
func SavePick(label, value string) error {
	raw := LoadPickHistory()
	raw[label] = map[string]interface{}{"last": value}
	return writeHistory(raw)
}

// writeHistory 将记忆写回 history.toml（自动建目录）。
func writeHistory(raw map[string]interface{}) error {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s history — last selection per key (auto-managed)\n", Name()))
	if err := toml.NewEncoder(&buf).Encode(raw); err != nil {
		return fmt.Errorf("编码 history.toml 失败: %w", err)
	}
	p := PickHistoryPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", p, err)
	}
	return nil
}
