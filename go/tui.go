// tui — 过滤选择器 TUI 基础组件。
//
// 提供带实时输入过滤的候选选择器（bubbletea），被 pick / _menu 复用：
// 标题栏 / 分隔线 / 列表（光标行绿色粗体 + 命中高亮）/ 输入行（闪烁光标）/ 状态栏。
//
// 布局（自上而下）：
//
//	标题栏
//	──── 分隔线 ────
//	列表（视口滚动）
//	──── 分隔线 ────
//	❯ 输入行▏
//	2/5 · 快捷键提示
//
// 渲染与按键唯一实现在引擎（本包），任何语言宿主经协议子命令拿到同样的
// 视觉与交互效果；绑定层禁止自绘或重实现匹配。
package picktui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// Options 配置过滤选择器 TUI。
type Options struct {
	Title      string        // 标题栏主文本（如 "pick" 命令的 "<name> pick"）
	Subtitle   string        // 标题栏次要文本（灰色，可选）
	Cands      []Candidate   // 全量候选（Value 为选中值，Desc 为展示用描述）
	Query      string        // 预填查询
	FilterOpts FilterOptions // 匹配模式（子串 AND / token 前缀 / 模糊）
	Initial    int           // 初始光标（默认 0）
	JumpKeys   bool          // 允许数字 1-9 直选（无输入时生效；输入后数字作为过滤字符）
	StatusHint string        // 底部快捷键提示
	MemoryKey  string        // 选择记忆 key（非空且 query 为空时生效：初始光标停在上次选中，选中后回写）
	Multi      bool          // 多选模式：space 切换勾选，enter 确认全部勾选项
	Selected   []string      // 多选模式初始勾选的 Value 列表
}

// Result 是选择器的返回结果。
type Result struct {
	Value     string   // 单选模式：选中值
	Selected  []string // 多选模式：勾选的 Value 列表（按 Cands 顺序）
	Index     int
	Cancelled bool // esc/ctrl+c 取消，或过滤结果为空
}

// Run 在 tty 上运行选择器并返回结果。
//
// 调用方负责打开 /dev/tty（渲染与输入都走它，stdout 留给调用方输出选中值）
// 并处理非交互退化；本函数不做 TTY 检测。
func Run(opts Options, tty *os.File) (Result, error) {
	// 强制 TrueColor，绕开 lipgloss 基于 os.Stdout 的自动探测：
	// 调用方（如 wrapper 的 $(pick ...)）stdout 常被命令替换捕获成管道，
	// 自动探测会判定为无色。SetColorProfile 在原地修改单例 renderer 的字段
	// （不换指针），因此包初始化时已创建的 Style 也能立即感知到颜色档位。
	lipgloss.SetColorProfile(termenv.TrueColor)

	// 选择记忆：仅 query 为空时应用（有预填过滤词说明是精确搜索，不记忆）。
	initial := opts.Initial
	if opts.MemoryKey != "" && opts.Query == "" {
		if last := LastPick(LoadPickHistory(), opts.MemoryKey); last != "" {
			for i, c := range opts.Cands {
				if c.Value == last {
					initial = i
					break
				}
			}
		}
	}

	m := newModel(opts, initial)
	p := tea.NewProgram(m, tea.WithInput(tty), tea.WithOutput(tty))
	final, err := p.Run()
	if err != nil {
		return Result{}, err
	}
	res := final.(model)
	if res.cancelled || len(res.filtered) == 0 {
		return Result{Cancelled: true}, nil
	}
	if opts.Multi {
		return Result{Selected: res.selectedValues(opts.Cands), Index: res.cursor}, nil
	}
	chosen := res.filtered[res.cursor].Value
	if opts.MemoryKey != "" {
		if err := SavePick(opts.MemoryKey, chosen); err != nil {
			fmt.Fprintf(os.Stderr, "%s: 记录选择记忆失败: %v\n", Name(), err)
		}
	}
	return Result{Value: chosen, Index: res.cursor}, nil
}

// ---- 样式 ----

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("120")) // 标题（绿粗体）
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))            // 次要文本（灰）
	okStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true)  // 计数（青绿）
	hlStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("213"))            // 命中片段（粉色，非光标行）
	curStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("120")) // 光标行：整行绿色粗体
	sepStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("237"))            // 分隔线（浅灰）
	promptStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("120")) // 输入提示符 ❯
)

// tickMsg 驱动输入光标闪烁（每 500ms 一次）。
type tickMsg time.Time

// model 是过滤选择器的 bubbletea 模型。
// 手写文本输入（append-only query + backspace/ctrl+u/ctrl+w）与视口滚动，无 bubbles 依赖。
type model struct {
	opts      Options
	filtered  []Candidate // 当前过滤结果
	query     string      // 查询字符串（append-only，光标始终在末尾）
	cursor    int         // 列表光标位置
	scrollY   int         // 视口滚动位置（首个可见行索引）
	height    int         // 终端高度（tea.WindowSizeMsg）
	width     int         // 终端宽度（tea.WindowSizeMsg），用于分隔线与反色行延伸
	phase     int         // 光标闪烁相位（0=显示 ▏，1=隐藏）
	cancelled bool
	selected  map[string]bool // 多选模式勾选集合（Value → true）
}

func newModel(opts Options, initial int) model {
	m := model{opts: opts, query: opts.Query, cursor: initial}
	if m.cursor < 0 || m.cursor >= len(opts.Cands) {
		m.cursor = 0
	}
	if opts.Multi {
		m.selected = make(map[string]bool, len(opts.Selected))
		for _, v := range opts.Selected {
			m.selected[v] = true
		}
	}
	m.recompute()
	return m
}

// selectedValues 按 Cands 顺序返回勾选的 Value 列表（多选模式）。
func (m model) selectedValues(cands []Candidate) []string {
	if len(m.selected) == 0 {
		return nil
	}
	var out []string
	for _, c := range cands {
		if m.selected[c.Value] {
			out = append(out, c.Value)
		}
	}
	return out
}

// recompute 根据当前 query 重新过滤，保持光标在有效范围内。
func (m *model) recompute() {
	m.filtered = FilterStructuredCandidates(m.opts.Cands, m.query, m.opts.FilterOpts)
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.ensureVisible()
}

func (m model) Init() tea.Cmd {
	return tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.width = msg.Width
		m.ensureVisible()
		return m, nil
	case tickMsg:
		m.phase ^= 1
		return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return tickMsg{} })
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if len(m.filtered) > 0 {
				return m, tea.Quit
			}
		case " ":
			// 多选模式：无过滤输入时空格勾选/取消（与数字直选同规则——已有输入时
			// 空格作为过滤字符）；单选模式空格始终是过滤字符：多关键字查询
			//（如 `ele dev`）需要能输入空格才能拆词过滤。
			if m.opts.Multi && m.query == "" && m.cursor < len(m.filtered) {
				v := m.filtered[m.cursor].Value
				if m.selected[v] {
					delete(m.selected, v)
				} else {
					m.selected[v] = true
				}
			} else {
				m.query += " "
				m.recompute()
			}
		case "up", "k", "ctrl+p":
			// 循环移动：首项按上键回到末项（视口同步跳到末尾）
			if n := len(m.filtered); n > 1 {
				m.cursor = (m.cursor - 1 + n) % n
				m.ensureVisible()
			}
		case "down", "j", "ctrl+n", "tab":
			// 循环移动：末项按下键回到首项
			if n := len(m.filtered); n > 1 {
				m.cursor = (m.cursor + 1) % n
				m.ensureVisible()
			}
		// 数字键 1-9 直选（菜单用）：无输入时跳过逐行移动快速选中；
		// 已输入过滤关键字时数字作为普通字符参与过滤。
		case "1", "2", "3", "4", "5", "6", "7", "8", "9":
			// 数字 1-9 直选仅在开启 JumpKeys（菜单）且无输入时生效；
			// 否则数字作为普通字符参与过滤。
			if m.opts.JumpKeys && m.query == "" {
				n := int(msg.String()[0] - '0')
				if n-1 < len(m.filtered) {
					m.cursor = n - 1
					return m, tea.Quit
				}
			} else {
				m.query += msg.String()
				m.recompute()
			}
		case "backspace":
			if len(m.query) > 0 {
				r := []rune(m.query)
				m.query = string(r[:len(r)-1])
				m.recompute()
			}
		case "ctrl+u":
			m.query = ""
			m.recompute()
		case "ctrl+w":
			fields := strings.Fields(m.query)
			if len(fields) > 0 {
				m.query = strings.Join(fields[:len(fields)-1], " ")
			} else {
				m.query = ""
			}
			m.recompute()
		default:
			s := msg.String()
			if isPrintableKey(s) {
				m.query += s
				m.recompute()
			}
		}
	}
	return m, nil
}

// isPrintableKey 判断 bubbletea KeyMsg 字符串是否为可打印字符（单 rune，非控制键）。
func isPrintableKey(s string) bool {
	runes := []rune(s)
	if len(runes) != 1 {
		return false
	}
	r := runes[0]
	return r >= 32 && r != 127
}

// ensureVisible 调整视口使光标可见。
// 固定行数：标题(1) + 分隔线(1) + [列表] + 分隔线(1) + 输入行(1) + 状态栏(1) = 5
func (m *model) ensureVisible() {
	const fixedLines = 5
	visibleRows := m.height - fixedLines
	if visibleRows <= 0 {
		visibleRows = 10 // WindowSizeMsg 尚未到达时的默认值
	}
	if m.cursor < m.scrollY {
		m.scrollY = m.cursor
	}
	if m.cursor >= m.scrollY+visibleRows {
		m.scrollY = m.cursor - visibleRows + 1
	}
}

func (m model) View() string {
	const fixedLines = 5
	visibleRows := m.height - fixedLines
	if visibleRows <= 0 {
		visibleRows = 10
	}
	width := m.width
	if width <= 0 {
		width = 80
	}
	sep := sepStyle.Render(strings.Repeat("─", width))

	var b strings.Builder
	// 标题栏
	b.WriteString(titleStyle.Render(m.opts.Title))
	if m.opts.Subtitle != "" {
		b.WriteString(dimStyle.Render("  ·  " + m.opts.Subtitle))
	}
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")

	// 列表（视口区间）
	end := m.scrollY + visibleRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := m.scrollY; i < end; i++ {
		c := m.filtered[i]
		box := ""
		if m.opts.Multi {
			if m.selected[c.Value] {
				box = okStyle.Render("[x] ")
			} else {
				box = "[ ] "
			}
		}
		if i == m.cursor {
			// 光标行：整行绿色粗体（不再嵌套命中高亮，避免内层 \x1b[0m 重置外层颜色）
			line := "  ▸ " + box + c.Value
			if c.Desc != "" {
				line += "  " + c.Desc
			}
			if pad := width - lipgloss.Width(line); pad > 0 {
				line += strings.Repeat(" ", pad)
			}
			b.WriteString(curStyle.Render(line))
		} else {
			b.WriteString("    " + box + renderCandidateLine(c, m.query, m.opts.FilterOpts, hlStyle, width))
		}
		b.WriteString("\n")
	}
	if len(m.filtered) == 0 {
		b.WriteString(dimStyle.Render("    (no matches)"))
		b.WriteString("\n")
	}

	// 分隔线 + 输入行 + 状态栏
	b.WriteString(sep)
	b.WriteString("\n")
	cursorBlock := "▏"
	if m.phase == 1 {
		cursorBlock = " "
	}
	b.WriteString(promptStyle.Render("❯ ") + m.query + cursorBlock)
	b.WriteString("\n")
	count := fmt.Sprintf("%d/%d", len(m.filtered), len(m.opts.Cands))
	b.WriteString("  " + okStyle.Render(count))
	if m.opts.StatusHint != "" {
		b.WriteString(dimStyle.Render("  ·  " + m.opts.StatusHint))
	}
	b.WriteString("\n")

	return b.String()
}

// renderCandidateLine 渲染候选行：value 命中高亮 + 灰色描述（按显示宽度截断）。
// cursor=true 时描述直接叠加在反色块上（不额外着色，保持绿底黑字可读性）。
func renderCandidateLine(c Candidate, query string, opts FilterOptions, hlStyle lipgloss.Style, width int) string {
	value := renderHighlighted(c.Value, query, opts, hlStyle)
	if c.Desc == "" {
		return value
	}
	// 描述仅占 value 之后的剩余宽度（减 2 个分隔空格），太窄就不展示
	remaining := width - lipgloss.Width(value) - 2
	if remaining < 4 {
		return value
	}
	return value + "  " + dimStyle.Render(truncateWidth(c.Desc, remaining))
}

// truncateWidth 按显示宽度截断 s 至 maxWidth，超长末尾补 …。
func truncateWidth(s string, maxWidth int) string {
	if maxWidth <= 1 {
		return "…"
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	w := 0
	for _, r := range runes {
		rw := lipgloss.Width(string(r))
		if w+rw > maxWidth-1 {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String() + "…"
}

// renderHighlighted 渲染候选字符串，高亮匹配片段（hlStyle 决定命中片段的样式：
// 光标行传入同色粗体，普通行传入粉色）。
func renderHighlighted(s, query string, opts FilterOptions, hlStyle lipgloss.Style) string {
	ranges := HighlightRanges(s, query, opts)
	if len(ranges) == 0 {
		return s
	}
	runes := []rune(s)
	var b strings.Builder
	pos := 0
	for _, r := range ranges {
		if pos < r[0] {
			b.WriteString(string(runes[pos:r[0]]))
		}
		b.WriteString(hlStyle.Render(string(runes[r[0]:r[1]])))
		pos = r[1]
	}
	if pos < len(runes) {
		b.WriteString(string(runes[pos:]))
	}
	return b.String()
}
