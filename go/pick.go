// Pick — 动态候选关键字过滤选择器命令。
//
// 面向候选多、长且相似的场景（如 git branch 输出的长分支名）。三种匹配模式
// （子串AND / token前缀 / 模糊子序列），多来源候选（位置参数 / stdin / --from），
// bubbletea TUI 实时过滤 + 高亮 + 视口滚动。非 TTY 退化打印首个候选。
//
// 结构化候选约定：来源输出每行按 key<TAB>description 拆分为"选中值 + 展示描述"，
// 无 TAB 的行整行作为 key。过滤只匹配 key，TUI 展示 description，选中输出 key——
// 让"展示与选中分离"。取数与转换分离：--from <cmd> 只取原始候选输出，--map <cmd>
// 作为转换器接收其 stdout（或 stdin）到自身 stdin，输出 key<TAB>des 结构化行，
// 解析逻辑无需再手拼进 --from。外部解析器（node / python / awk 等）皆可：
//
//	picktui pick --from 'npm run' --map 'node ~/.config/picktui/pickers/npm-run.mjs'
//
// 也作为扫描宿主（如 shell wrapper）的底层引擎：缩写命中 pick 源时通过
// <name> pick --from <cmd> --label <path> 调起本命令。
//
// 品牌与约定皆可对齐宿主：SetName（消息前缀/TUI 标题）、SetOffEnv（非交互
// 开关环境变量）、SetDataDir（记忆/确认文件目录）、SetMapResolver（--map 裸名
// 解析钩子）。退出码：0 成功 / 1 运行错误 / 2 用法错误 / 130 取消。
package picktui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// openDevTTY 打开 /dev/tty 供 TUI 渲染与输入（stdout 留给选中值）。
func openDevTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// Pick 实现 `pick [flags] [cand...]`，返回进程退出码。
// out 接收选中值（数据，不带色码），errw 接收错误/警告消息：
//
//	usage:
//	  pick [flags] [cand...]
//	  git branch | pick
//	  pick --from "git branch"
//
//	flags:
//	  -q, --query <text>   预填查询；非 TTY 时过滤后打印首个匹配
//	  --from <cmd>         候选来源：执行命令取 stdout
//	  --map <cmd>          转换器：接收 --from（或 stdin）输出，产出 key<TAB>des
//	  --sep <chars>        token 前缀模式，分隔符集合（默认 _，可多字符如 ':_'）
//	  --fuzzy              子序列模糊匹配
//	  --label <name>       选择记忆键，复用 history.toml
//	  -1, --select-1       仅一个匹配时自动选中（跳过 TUI）
//	  --auto               配合 -q：精确或唯一前缀匹配直接输出选中值（如 npm run re →
//	                      release）；首次匹配该 (label, value) 需用户确认（记录于
//	                      confirm.toml，之后不再询问；<OffEnv>=off/无 tty/无 label
//	                      跳过确认），拒绝确认回退 TUI 改选；无法唯一确定时回退
//	                      TUI（query 预填过滤）
func Pick(args []string, out, errw io.Writer) int {
	return pickWith(args, out, errw, openDevTTY)
}

// pickWith 是 Pick 的可注入实现（openTTY 便于测试非交互退化路径）。
func pickWith(args []string, out, errw io.Writer, openTTY func() (*os.File, error)) int {
	var query, from, mapper, label, sep string
	var fuzzy, select1, auto bool
	var rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "-q" || a == "--query":
			if i+1 < len(args) {
				i++
				query = args[i]
			}
		case strings.HasPrefix(a, "--query="):
			query = strings.TrimPrefix(a, "--query=")
		case a == "--from":
			if i+1 < len(args) {
				i++
				from = args[i]
			}
		case strings.HasPrefix(a, "--from="):
			from = strings.TrimPrefix(a, "--from=")
		case a == "--map":
			if i+1 < len(args) {
				i++
				mapper = args[i]
			}
		case strings.HasPrefix(a, "--map="):
			mapper = strings.TrimPrefix(a, "--map=")
		case a == "--label":
			if i+1 < len(args) {
				i++
				label = args[i]
			}
		case strings.HasPrefix(a, "--label="):
			label = strings.TrimPrefix(a, "--label=")
		case a == "--sep":
			if i+1 < len(args) {
				i++
				sep = args[i]
			}
		case strings.HasPrefix(a, "--sep="):
			sep = strings.TrimPrefix(a, "--sep=")
		case a == "--fuzzy":
			fuzzy = true
		case a == "-1" || a == "--select-1":
			select1 = true
		case a == "--auto":
			auto = true
		default:
			rest = append(rest, a)
		}
	}

	// --map 裸名在执行时经 SetMapResolver 钩子展开（如 "npm-run" → node <绝对路径>）
	if mapper != "" {
		mapper = resolveMapArg(mapper)
	}

	// 构造 FilterOptions
	opts := FilterOptions{Mode: MatchSubstring}
	if fuzzy {
		opts.Mode = MatchFuzzy
	} else if sep != "" {
		opts.Mode = MatchTokenPrefix
		opts.Sep = sep
	}

	// --map 只做"来源 → 转换"，与位置参数互斥
	if mapper != "" && len(rest) > 0 {
		errln(errw, fmt.Sprintf("%s pick: --map 只与 --from 或 stdin 配合，不能与位置参数同用", Name()))
		return 2
	}

	// 获取候选（优先级：--from+--map > --from > 位置参数 > stdin+--map > stdin），统一解析为结构化候选
	var cands []Candidate
	if from != "" && mapper != "" {
		c, err := StructuredFromCommandMapped(from, mapper)
		if err != nil {
			renderMapError(from, mapper, err, errw)
			return 1
		}
		cands = c
	} else if from != "" {
		c, err := StructuredFromCommand(from)
		if err != nil {
			errf(errw, "%s pick: %s: %v\n", Name(), from, err)
			return 1
		}
		cands = c
	} else if len(rest) > 0 {
		cands = StructuredCandidates(rest)
	} else if mapper != "" && !isTerminal(os.Stdin) {
		c, err := StructuredFromReaderMapped(os.Stdin, mapper)
		if err != nil {
			renderMapError("", mapper, err, errw)
			return 1
		}
		cands = c
	} else if !isTerminal(os.Stdin) {
		cands = StructuredFromReader(os.Stdin)
	}
	if len(cands) == 0 {
		errln(errw, fmt.Sprintf("%s pick: no candidates", Name()))
		return 1
	}

	// --auto 自动选中：query 精确命中或唯一前缀命中时直接输出（如 npm run re →
	// release），跳过 TUI。首次匹配该 (label, value) 需用户确认，防止自动匹配
	// 执行了用户并不想执行的操作；确认记录到 confirm.toml，之后同解析不再询问。
	// 拒绝确认则回退下方 TUI（query 预填）让用户改选或取消。
	// <OffEnv>=off / 打不开 /dev/tty / 无 label 时跳过确认（脚本确定性语义，
	// 且无 label 无法记录状态）。无法唯一确定（无匹配/多前缀）时回退常规流程。
	if auto && query != "" {
		if v, ok := AutoResolve(cands, query); ok {
			if label == "" || IsConfirmed(label, v) || os.Getenv(OffEnv()) == "off" {
				pickSaveOutput(label, v, out, errw)
				return 0
			}
			if tty, err := openTTY(); err == nil {
				ok := confirmAutoPrompt(tty, label, v)
				tty.Close()
				if ok {
					if err := Confirm(label, v); err != nil {
						errf(errw, "%s: 记录确认状态失败: %v\n", Name(), err)
					}
					pickSaveOutput(label, v, out, errw)
					return 0
				}
				// 用户拒绝确认：回退到下方 TUI 选择（query 预填过滤）
			} else {
				// 无 /dev/tty：非交互环境跳过确认，保持确定性执行
				pickSaveOutput(label, v, out, errw)
				return 0
			}
		}
	}

	// -1 自动选中
	if select1 {
		filtered := FilterStructuredCandidates(cands, query, opts)
		if len(filtered) == 1 {
			if label != "" {
				if err := SavePick(label, filtered[0].Value); err != nil {
					errf(errw, "%s: 记录选择记忆失败: %v\n", Name(), err)
				}
			}
			outln(out, filtered[0].Value)
			return 0
		}
	}

	// <OffEnv>=off 或无 /dev/tty → 非交互模式
	if os.Getenv(OffEnv()) == "off" {
		return pickNonInteractive(cands, query, opts, label, auto, out, errw)
	}
	tty, err := openTTY()
	if err != nil {
		return pickNonInteractive(cands, query, opts, label, auto, out, errw)
	}
	defer tty.Close()

	res, err := Run(Options{
		Title:      Name() + " pick",
		Subtitle:   label,
		Cands:      cands,
		Query:      query,
		FilterOpts: opts,
		StatusHint: "type to filter · ↑↓ move · enter select · esc cancel",
		MemoryKey:  label,
	}, tty)
	if err != nil {
		errln(errw, fmt.Sprintf("%s: %v", Name(), err))
		return 1
	}
	if res.Cancelled {
		return 130
	}
	// 自动解析流程（--auto）中用户从 TUI 显式选中 → 视为对该解析的确认：
	// 记录后，下次自动匹配同 (label, value) 不再询问（拒绝自动匹配后改选的
	// 场景同样受益——手动选过即代表接受）。
	if auto && label != "" {
		if err := Confirm(label, res.Value); err != nil {
			errf(errw, "%s: 记录确认状态失败: %v\n", Name(), err)
		}
	}
	outln(out, res.Value)
	return 0
}

// confirmAutoPrompt 在 tty 上询问用户是否确认首次自动匹配的解析结果。
// 返回 true=确认；读取失败或非 y/yes 一律视为拒绝（保守默认，宁可多问一次）。
// stdout 可能被宿主 shell wrapper 的命令替换捕获，因此提示与读取都走 /dev/tty。
func confirmAutoPrompt(tty *os.File, label, value string) bool {
	fmt.Fprintf(tty, "\r\n%s: 首次自动匹配 %s → %s，确认执行？[y/N] ", Name(), label, value)
	line, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return false
	}
	return ParseConfirmAnswer(line)
}

// ParseConfirmAnswer 解析确认回答：y/yes（大小写不敏感，容忍首尾空白）为确认。
// 导出以支持测试与宿主复用。
func ParseConfirmAnswer(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "y", "yes":
		return true
	}
	return false
}

// pickSaveOutput 记录选择记忆（label 非空时）并输出选中值（数据输出，不带色码）。
func pickSaveOutput(label, v string, out, errw io.Writer) {
	if label != "" {
		if err := SavePick(label, v); err != nil {
			errf(errw, "%s: 记录选择记忆失败: %v\n", Name(), err)
		}
	}
	outln(out, v)
}

// pickNonInteractive 非交互模式：无 query 打印首个候选（确定性脚本语义）；
// 有 query 时过滤后取首个匹配，无匹配则报错返回非零——绝不静默回退到首个候选，
// 避免脚本管道在查询打错时拿到毫不相干的选中值。
// auto 为 true 时先做精确/唯一前缀解析：能唯一确定才输出，否则报错（不取首个匹配）。
func pickNonInteractive(cands []Candidate, query string, opts FilterOptions, label string, auto bool, out, errw io.Writer) int {
	if auto && query != "" {
		if v, ok := AutoResolve(cands, query); ok {
			pickSaveOutput(label, v, out, errw)
			return 0
		}
		errf(errw, "%s pick: no unique match for %q\n", Name(), query)
		return 1
	}
	if query == "" {
		chosen := cands[0].Value
		if label != "" {
			if err := SavePick(label, chosen); err != nil {
				errf(errw, "%s: 记录选择记忆失败: %v\n", Name(), err)
			}
		}
		outln(out, chosen)
		return 0
	}
	filtered := FilterStructuredCandidates(cands, query, opts)
	if len(filtered) == 0 {
		errf(errw, "%s pick: no match for %q\n", Name(), query)
		return 1
	}
	chosen := filtered[0].Value
	if label != "" {
		if err := SavePick(label, chosen); err != nil {
			errf(errw, "%s: 记录选择记忆失败: %v\n", Name(), err)
		}
	}
	outln(out, chosen)
	return 0
}

// renderMapError 渲染 --map 失败信息。脚本文件缺失（MapperScriptError）时给出
// 明确指引而非透传解释器堆栈；其余错误保持原有格式（含 stderr 诊断）。
func renderMapError(from, mapper string, err error, errw io.Writer) {
	head := fmt.Sprintf("%s pick: --map %s", Name(), mapper)
	if from != "" {
		head = fmt.Sprintf("%s pick: %s | %s", Name(), from, mapper)
	}
	var mse *MapperScriptError
	if errors.As(err, &mse) {
		errln(errw, head+":")
		errf(errw, "  %v\n", mse)
		errln(errw, "  → 脚本可能已被移动、删除或重命名")
		errln(errw, "  → 修复: 重新执行以下命令重新绑定:")
		errln(errw, fmt.Sprintf("    `%s add <cmd> --pick '<cmd>' --map '<新命令>'`", Name()))
		return
	}
	errf(errw, "%s: %v\n", head, err)
}
