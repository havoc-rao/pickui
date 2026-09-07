// Package pickui：可供复用的 TUI 辅助候选选择器（fzf 风格）。
//
// 核心能力：
//   - tui：bubbletea 实时过滤选择器（标题/分隔/列表/输入/状态栏，视口滚动、
//     命中高亮、数字直选、多选、选择记忆），渲染与输入都走调用方传入的 tty，
//     stdout 留给选中值——适合 $(cmd) 命令替换场景；
//   - filter：候选匹配引擎（子串 AND / token 前缀 / 子序列模糊）+ 命中区间
//     高亮 + --auto 精确/唯一前缀自动解析；
//   - cand：候选来源（位置参数 / 标准输入 / 执行命令 / --map 转换器）与
//     结构化候选 key<TAB>description 解析清洗；
//   - history/confirm：选择记忆与自动匹配首次确认（TOML 存储，目录可配置）；
//   - pick/menu：完整 CLI 命令（Pick/Menu），可嵌入其他 CLI，也提供独立
//     二进制 cmd/pickui。
//
// 嵌入时可通过 SetName / SetOffEnv / SetDataDir / SetMapResolver 对齐宿主
// 项目的品牌、非交互环境变量与状态文件目录（shr 即以此方式嵌入）。
package pickui

import (
	"os"
	"path/filepath"
)

// ---- 可配置项（包级，进程内一次性设置） ----

var (
	cmdName     = "pickui"          // 提示/错误前缀中的命令名
	offEnv      = "PICKUI_PICK"     // 非交互开关环境变量（值 "off" 时跳过 TUI）
	dataDir     = ""                // 显式数据目录（空 = 按默认规则解析）
	mapResolver func(string) string // --map 值解析钩子（默认原样返回）
)

// SetName 设置提示/错误前缀中的命令名（默认 "pickui"）。shr 嵌入时设为 "shr"，
// 消息即与旧版完全一致（如 "shr pick: no candidates"）。
func SetName(name string) { cmdName = name }

// Name 返回当前命令名。
func Name() string { return cmdName }

// SetOffEnv 设置非交互开关的环境变量名；值为 "off" 时 pick/menu 跳过 TUI
// （确定性脚本语义）。默认 "PICKUI_PICK"；shr 嵌入时设为 "SHR_PICK"。
func SetOffEnv(name string) { offEnv = name }

// OffEnv 返回当前非交互开关环境变量名。
func OffEnv() string { return offEnv }

// SetDataDir 设置状态文件目录（history.toml / confirm.toml）。
// 默认解析优先级：$PICKUI_CONFIG_DIR → $XDG_CONFIG_HOME/pickui → ~/.config/pickui。
// shr 嵌入时设为 shr 自己的配置目录，保持既有记忆不丢失。
func SetDataDir(dir string) { dataDir = dir }

// DataDir 返回状态文件目录。
func DataDir() string {
	if dataDir != "" {
		return dataDir
	}
	if d := os.Getenv("PICKUI_CONFIG_DIR"); d != "" {
		return d
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, _ := os.UserHomeDir()
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "pickui")
}

// SetMapResolver 设置 --map 值的解析钩子：宿主可在执行前把裸名（如
// "npm-run"）展开为完整命令（如 "node /abs/path/npm-run.mjs"）。
// 默认原样返回。
func SetMapResolver(fn func(string) string) { mapResolver = fn }

// resolveMapArg 应用 map 解析钩子（无钩子时原样返回）。
func resolveMapArg(m string) string {
	if mapResolver != nil {
		return mapResolver(m)
	}
	return m
}
