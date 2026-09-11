// Menu — 多值缩写运行时 TUI 菜单命令。
//
// 由宿主 shell wrapper 的函数回调：
//
//	picktui _menu "<label>" <cand1> <cand2> ...
//
// TUI 渲染到 /dev/tty（不被 $(...) 命令替换捕获），选中候选写到 out 供
// wrapper 捕获后 command 执行；取消则返回非零（130），wrapper 据退出码中止执行。
// 无 /dev/tty（如脚本/管道）退化为取首个候选并警告，避免卡住非交互执行。
//
// 交互：输入过滤（子串 AND）、高亮、视口滚动，另保留菜单特有的数字键 1-9 直选
// （无输入时生效；输入后数字作为过滤字符）。MemoryKey 即 label。
package picktui

import (
	"fmt"
	"io"
	"os"
)

// MenuUsage 返回菜单命令的用法文本（随 SetName 对齐）。
func MenuUsage() string {
	return fmt.Sprintf("usage: %s _menu <label> <cand...>", Name())
}

// Menu 实现 `_menu <label> <cand...>`，返回进程退出码。
// out 接收选中值（数据，不带色码），errw 接收错误/警告消息。
func Menu(args []string, out, errw io.Writer) int {
	return menuWith(args, out, errw, openDevTTY)
}

// menuWith 是 Menu 的可注入实现（openTTY 便于测试非交互退化路径）。
func menuWith(args []string, out, errw io.Writer, openTTY func() (*os.File, error)) int {
	if len(args) < 2 {
		errln(errw, MenuUsage())
		return 2
	}
	label := args[0]
	cands := args[1:]

	// TUI 必须有 /dev/tty：渲染与输入都走它，out 留给选中值。
	// 无 /dev/tty（非交互）→ 取首个候选并警告，保证脚本不卡。
	tty, err := openTTY()
	if err != nil {
		errf(errw, "%s: 多值缩写 %q 在非交互环境下取首个候选 %q\n", Name(), label, cands[0])
		outln(out, cands[0])
		return 0
	}
	defer tty.Close()

	res, err := Run(Options{
		Title:      Name() + " " + label,
		Subtitle:   "多值缩写",
		Cands:      StringCandidates(cands),
		FilterOpts: FilterOptions{Mode: MatchSubstring},
		JumpKeys:   true,
		StatusHint: "type to filter · ↑↓ move · 1-9 jump · enter confirm · esc cancel",
		MemoryKey:  label, // 记住上次选择，下次默认停在此处
	}, tty)
	if err != nil {
		errln(errw, fmt.Sprintf("%s: %v", Name(), err))
		return 1
	}
	if res.Cancelled {
		return 130 // 类似 SIGINT 退出码，wrapper 会 return 中止
	}
	outln(out, res.Value)
	return 0
}
