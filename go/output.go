// output — Pick/Menu 的消息输出出口。
//
// 约定：选中值（数据）直写 out，绝不染色；错误/警告消息写 errw，行首的
// "<命令名> <子命令>:" 前缀在流为交互式终端时着红色粗体，帮助快速定位来源。
// 管道/重定向/脚本消费时自动降级为纯文本，保证 `picktui pick | grep`、
// eval "$(picktui pick ...)" 等场景输出不受污染。
package picktui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

const (
	ansiRed    = "1;38;5;196" // 错误前缀（红粗体）
	ansiYellow = "1;38;5;228" // 引导/警告（黄粗体，预留）
)

// colorize 用 ANSI 256 色码包裹 s。
func colorize(code, s string) string {
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

// isTerminal 判断文件描述符是否为交互式终端（用 golang.org/x/term 的 ioctl 检测，
// 比 os.ModeCharDevice 更准确——后者会把 /dev/null 这类字符设备误判为终端）。
func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// errPrefixLen 返回 s 行首 "<命令名>( <子命令>)?" 前缀的长度（冒号含在内），
// 不匹配时返回 0。子命令为 [a-z_]+（如 pick / _menu / hist get 命中第一段）。
func errPrefixLen(name, s string) int {
	if !strings.HasPrefix(s, name) {
		return 0
	}
	rest := s[len(name):]
	if rest == "" || rest[0] != ' ' && rest[0] != ':' {
		return 0
	}
	if rest[0] == ':' {
		return len(name) + 1
	}
	i := 1
	for i < len(rest) && isWordChar(rest[i]) {
		i++
	}
	if i > 1 && i < len(rest) && rest[i] == ':' {
		return len(name) + i + 1
	}
	return 0
}

func isWordChar(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-'
}

// paintErrLine 给单行（不含换行）的错误消息着色：行首 <name>...: 前缀红粗体。
// 非终端或无可着色前缀原样返回。
func paintErrLine(f *os.File, line string) string {
	if !isTerminal(f) || line == "" {
		return line
	}
	if n := errPrefixLen(Name(), line); n > 0 {
		return colorize(ansiRed, line[:n]) + line[n:]
	}
	return line
}

// emitErr 将 s 逐行着色后写入 errw（保留原换行）。errw 若不是 *os.File
// （或不是终端），原样输出。
func emitErr(errw io.Writer, s string) {
	var f *os.File
	if f2, ok := errw.(*os.File); ok {
		f = f2
	}
	var b strings.Builder
	for _, line := range strings.SplitAfter(s, "\n") {
		if line == "" {
			continue
		}
		body := strings.TrimSuffix(line, "\n")
		if f != nil {
			body = paintErrLine(f, body)
		}
		b.WriteString(body)
		if line != body {
			b.WriteByte('\n')
		}
	}
	fmt.Fprint(errw, b.String())
}

// errf / errln 是 Pick/Menu 错误消息的统一出口：语义前缀着色（TTY 下），
// 管道/重定向时自动降级为纯文本。
func errf(errw io.Writer, format string, args ...any) { emitErr(errw, fmt.Sprintf(format, args...)) }
func errln(errw io.Writer, args ...any)               { emitErr(errw, fmt.Sprintln(args...)) }

// outln 直写数据行到 out（不带任何色码——选中值可能被 $(...) 消费）。
func outln(out io.Writer, args ...any) { fmt.Fprintln(out, args...) }
