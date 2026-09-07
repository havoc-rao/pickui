// cand — 候选来源与结构化解析。
//
// 结构化候选约定：源输出每行可按 key<TAB>description 拆分为"选中值 + 展示描述"，
// 无 TAB 的行整行作为 key。过滤只匹配 key，TUI 展示 description，选中输出 key——
// 让"展示与选中分离"（如 npm run 脚本列表带描述、git branch 带远端提示等）。
//
// 候选来源：位置参数 / 标准输入 / 执行命令（--from）/+ 转换器（--map）。
// 与过滤引擎（filter.go）同属纯函数层，不依赖 TUI，可独立单测。
package pickui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Candidate 是带可选描述的结构化候选：Value 为选中输出/执行的键，Desc 仅供展示。
type Candidate struct {
	Value string
	Desc  string
}

// CandidatesFromCommand 执行命令并返回清洗后的候选列表（10s 超时）。
// 失败时合并 stderr，让原因可诊断（如 git branch 在非仓库目录报 fatal）。
func CandidatesFromCommand(cmd string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := runCapture(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return CleanCandidates(strings.Split(string(out), "\n")), nil
}

// StructuredFromCommand 执行命令并返回结构化候选（key<TAB>des 解析），10s 超时。
// 失败时合并 stderr，让原因可诊断。
func StructuredFromCommand(cmd string) ([]Candidate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := runCapture(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return StructuredCandidates(strings.Split(string(out), "\n")), nil
}

// StructuredFromCommandMapped 取数 + 转换：执行 cmd 取其原始 stdout 作为 mapper 的 stdin，
// mapper 输出按结构化候选解析。两命令共享同一 10s 超时上下文。
// 用于分离"取数（--from）"与"转换（--map）"——--from 只负责来源，解析逻辑进 mapper，
// 无需再手拼 'from | parser' 管道（旧式内联管道写法仍兼容）。
func StructuredFromCommandMapped(cmd, mapper string) ([]Candidate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	out, err := runCapture(ctx, cmd)
	if err != nil {
		return nil, err
	}
	return runMapper(ctx, mapper, out)
}

// StructuredFromReaderMapped 从 reader 读原始输入作为 mapper 的 stdin，转换后解析结构化候选。
func StructuredFromReaderMapped(r io.Reader, mapper string) ([]Candidate, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return runMapper(ctx, mapper, data)
}

// runCapture 执行命令并返回原始 stdout；stderr 并入错误以便诊断（如 git 的 fatal）。
func runCapture(ctx context.Context, cmd string) ([]byte, error) {
	c := exec.CommandContext(ctx, "sh", "-c", cmd)
	var stderr strings.Builder
	c.Stderr = &stderr
	out, err := c.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%v: %s", err, msg)
		}
		return nil, err
	}
	return out, nil
}

// runMapper 执行 mapper，stdin 喂 input，输出按结构化候选解析；stderr 并入错误。
// 执行前先预检 mapper 引用的脚本文件，缺失时 fail-fast 返回 MapperScriptError，
// 避免 node/python 等解释器刷出冗长堆栈（常见于脚本被移动、删除或重命名）。
func runMapper(ctx context.Context, mapper string, input []byte) ([]Candidate, error) {
	if missing := mapperScriptMissing(mapper); missing != "" {
		return nil, &MapperScriptError{Path: missing}
	}
	m := exec.CommandContext(ctx, "sh", "-c", mapper)
	m.Stdin = bytes.NewReader(input)
	var stderr strings.Builder
	m.Stderr = &stderr
	out, err := m.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%v: %s", err, msg)
		}
		return nil, err
	}
	return StructuredCandidates(strings.Split(string(out), "\n")), nil
}

// MapperScriptError 报告 --map 命令中引用的脚本文件不存在（通常因脚本被移动、删除或重命名）。
// 宿主可用 errors.As 识别并给出修复指引，而非透传解释器堆栈。
type MapperScriptError struct {
	Path string
}

func (e *MapperScriptError) Error() string {
	return fmt.Sprintf("map 脚本文件不存在: %s", e.Path)
}

// mapperScriptMissing 扫描 mapper 命令，识别"解释器 + 脚本路径"形态，
// 返回第一个不存在的脚本文件路径；全部存在或无法识别返回空串。
// 保守策略：跳过 flags、引号、变量、管道等 shell 语法，只检查形如路径的 token
// （/ ./ ../ ~/ 前缀、带常见脚本扩展名），避免误拦 node -e / python3 -m 等合法用法。
func mapperScriptMissing(mapper string) string {
	fields := strings.Fields(mapper)
	if len(fields) < 2 {
		return ""
	}
	for _, tok := range fields[1:] { // 第一个 token 视为解释器，跳过
		if strings.HasPrefix(tok, "-") {
			continue
		}
		if strings.ContainsAny(tok, "'\"`${}|;&<>?*=") {
			continue
		}
		if !scriptPathLike(tok) {
			continue
		}
		p := expandTilde(tok)
		if _, err := os.Stat(p); err == nil {
			continue
		}
		return p
	}
	return ""
}

// scriptPathLike 判定 token 是否形如脚本文件路径：/ ./ ../ ~/ 前缀，
// 或带常见脚本解释器扩展名（含纯文件名，如 parser.mjs 在当前目录）。
func scriptPathLike(tok string) bool {
	if strings.HasPrefix(tok, "/") || strings.HasPrefix(tok, "./") ||
		strings.HasPrefix(tok, "../") || strings.HasPrefix(tok, "~/") {
		return true
	}
	return isScriptExt(filepath.Ext(tok))
}

// isScriptExt 常见脚本解释器可直接执行的文件扩展名。
func isScriptExt(ext string) bool {
	switch strings.ToLower(ext) {
	case ".mjs", ".cjs", ".js", ".ts", ".py", ".sh", ".bash", ".zsh", ".fish",
		".pl", ".rb", ".php", ".lua", ".exs", ".R":
		return true
	}
	return false
}

// expandTilde 展开 ~ 与 ~/ 前缀为用户主目录。
func expandTilde(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			if p == "~" {
				return h
			}
			return filepath.Join(h, p[2:])
		}
	}
	return p
}

// CandidatesFromReader 从 reader 逐行读取并清洗候选；cli 侧传 os.Stdin。
func CandidatesFromReader(r io.Reader) []string {
	return valuesOnly(StructuredFromReader(r))
}

// StructuredFromReader 从 reader 逐行读取并解析结构化候选；cli 侧传 os.Stdin。
func StructuredFromReader(r io.Reader) []Candidate {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return StructuredCandidates(lines)
}

func valuesOnly(cands []Candidate) []string {
	out := make([]string, len(cands))
	for i, c := range cands {
		out[i] = c.Value
	}
	return out
}

// CleanCandidates 预处理原始行：trim → 去 "* "/"+" 前缀 → 跳空行 → 去重保序。
// "* " 是 git branch 当前分支标记，"+ " 是 detached HEAD 标记。
// 与 StructuredCandidates 同源（TAB 拆分会合并描述），保持行为一致。
func CleanCandidates(lines []string) []string {
	return valuesOnly(StructuredCandidates(lines))
}

// StringCandidates 将纯字符串列表转为无描述的结构化候选（如 _menu 的静态候选）。
func StringCandidates(values []string) []Candidate {
	out := make([]Candidate, len(values))
	for i, v := range values {
		out[i] = Candidate{Value: v}
	}
	return out
}

// StructuredCandidates 预处理原始行得到结构化候选：
//
//  1. 先按 TAB 拆 key<TAB>description（无 TAB 则整行为 key）；
//  2. key 部分 trim → 去 "* "/"+ " 前缀（git branch 标记）→ 再 trim；
//  3. 跳空 key；按 key 去重保序——后到行可为先到同名 key 补缺失描述；
//  4. description 部分 trim（key 为空的行——如行首 TAB——直接跳过）。
func StructuredCandidates(lines []string) []Candidate {
	idx := map[string]int{}
	var out []Candidate
	for _, line := range lines {
		value, desc := line, ""
		if before, after, ok := strings.Cut(line, "\t"); ok {
			value, desc = before, after
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		value = strings.TrimPrefix(value, "* ")
		value = strings.TrimPrefix(value, "+ ")
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		desc = strings.TrimSpace(desc)
		if j, ok := idx[value]; ok {
			if out[j].Desc == "" && desc != "" {
				out[j].Desc = desc
			}
			continue
		}
		idx[value] = len(out)
		out = append(out, Candidate{Value: value, Desc: desc})
	}
	return out
}
