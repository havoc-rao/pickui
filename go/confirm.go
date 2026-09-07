// confirm — 自动匹配的首次用户确认记录（confirm.toml）。
//
// --auto 自动解析出候选时（如 `npm run re` → release），首次遇到该
// （label, value）解析需要用户确认，防止自动匹配执行了用户并不想执行的操作；
// 确认后记录到 confirm.toml，后续同解析不再询问。与 history.toml 一样独立于
// 宿主规则文件，避免写入时触发 wrapper 热加载。
//
// 文件格式（TOML，位于数据目录 confirm.toml）：
//
//	# pickui confirm — user-confirmed auto resolutions (auto-managed)
//	[confirmed]
//	"npm run" = ["release", "dev"]
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

// ConfirmFileName 自动匹配确认记录文件名，位于数据目录。
// 记录用户首次确认过的 (label, value) 解析，之后同解析不再询问。
const ConfirmFileName = "confirm.toml"

// ConfirmPath 返回确认记录文件路径。
func ConfirmPath() string {
	return filepath.Join(DataDir(), ConfirmFileName)
}

// LoadConfirmed 解码 confirm.toml 为 label → 已确认值列表。
// 文件不存在或损坏时返回空 map（静默忽略错误——确认记录只是安全辅助，
// 缺失时最多导致首次匹配再次询问，不影响主流程）。
func LoadConfirmed() map[string][]string {
	raw := map[string][]string{}
	if _, err := os.Stat(ConfirmPath()); err != nil {
		return raw
	}
	var doc struct {
		Confirmed map[string][]string `toml:"confirmed"`
	}
	if _, err := toml.DecodeFile(ConfirmPath(), &doc); err != nil || doc.Confirmed == nil {
		return map[string][]string{}
	}
	return doc.Confirmed
}

// IsConfirmed 报告 (label, value) 是否已被用户确认过。
// label 为空（无记忆键）时返回 false——调用方按"无法记录则跳过询问"处理。
func IsConfirmed(label, value string) bool {
	if label == "" {
		return false
	}
	for _, v := range LoadConfirmed()[label] {
		if v == value {
			return true
		}
	}
	return false
}

// Confirm 记录 (label, value) 为已确认（幂等：已存在则直接返回 nil）。
// 返回写入错误（如磁盘满、权限不足），调用方应记录但不必中断主流程。
func Confirm(label, value string) error {
	if label == "" {
		return nil
	}
	m := LoadConfirmed()
	for _, v := range m[label] {
		if v == value {
			return nil
		}
	}
	m[label] = append(m[label], value)
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("# %s confirm — user-confirmed auto resolutions (auto-managed)\n", Name()))
	if err := toml.NewEncoder(&buf).Encode(map[string]interface{}{"confirmed": m}); err != nil {
		return fmt.Errorf("编码 %s 失败: %w", ConfirmFileName, err)
	}
	p := ConfirmPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return fmt.Errorf("创建目录 %s 失败: %w", filepath.Dir(p), err)
	}
	if err := os.WriteFile(p, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败: %w", p, err)
	}
	return nil
}
