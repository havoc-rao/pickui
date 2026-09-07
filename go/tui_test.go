// Package pickui: 过滤选择器 TUI 模型的按键行为测试（内部包，直接驱动 model）。
package pickui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// keyMsg 构造可打印字符按键（KeyRunes 形式，与真实终端输入一致）。
func keyMsg(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// typeKeys 依次输入一串字符，返回最新 model。
func typeKeys(t *testing.T, m model, s string) model {
	t.Helper()
	for _, ch := range s {
		n, _ := m.Update(keyMsg(string(ch)))
		m = n.(model)
	}
	return m
}

func TestSpaceTypableInSingleSelect(t *testing.T) {
	m := newModel(Options{
		Cands: []Candidate{
			{Value: "electron:dev"},
			{Value: "electron:build"},
			{Value: "serve"},
		},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)

	// 输入 "ele dev"：空格必须能作为过滤字符输入（单选模式不勾选）
	m = typeKeys(t, m, "ele dev")
	if m.query != "ele dev" {
		t.Fatalf("query = %q, want %q", m.query, "ele dev")
	}
	// 多关键字子串 AND：ele dev → 命中 electron:dev
	if len(m.filtered) != 1 || m.filtered[0].Value != "electron:dev" {
		t.Fatalf("filtered = %v, want [electron:dev]", m.filtered)
	}
	// 空格不会改变选中集合（单选模式无勾选概念）
	if len(m.selected) != 0 {
		t.Fatalf("single-select should not toggle selection, selected=%v", m.selected)
	}
}

func TestSpaceToggleOnlyWhenQueryEmptyInMulti(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
		Multi:      true,
	}, 0)

	// 无过滤输入：空格勾选当前行
	n, _ := m.Update(keyMsg(" "))
	m = n.(model)
	if len(m.selected) != 1 || !m.selected["a"] {
		t.Fatalf("space should toggle selection with empty query, selected=%v", m.selected)
	}

	// 已有过滤输入：空格作为过滤字符（不勾选），与数字直选同规则
	m = typeKeys(t, m, "b ")
	if m.query != "b " {
		t.Fatalf("query = %q, want %q", m.query, "b ")
	}
	if len(m.selected) != 1 {
		t.Fatalf("space should not toggle while typing, selected=%v", m.selected)
	}
}

func TestSpaceKeywordFilteringAcrossColon(t *testing.T) {
	// 多关键字过滤在 token 前缀模式下同样命中冒号分隔的候选（ele → electron, dev → dev）
	m := newModel(Options{
		Cands: []Candidate{
			{Value: "electron:dev"},
			{Value: "electron:build"},
			{Value: "mydev"},
		},
		FilterOpts: FilterOptions{Mode: MatchTokenPrefix, Sep: ":_"},
	}, 0)

	m = typeKeys(t, m, "ele dev")
	if len(m.filtered) != 1 || m.filtered[0].Value != "electron:dev" {
		t.Fatalf("filtered = %v, want [electron:dev]", m.filtered)
	}
}

// ---- 光标循环移动（wrap-around） ----

// pressKey 向 model 发送单个按键并返回最新 model。
func pressKey(t *testing.T, m model, s string) model {
	t.Helper()
	n, _ := m.Update(keyMsg(s))
	return n.(model)
}

func TestCursorWrapsUpFromFirst(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}, {Value: "c"}, {Value: "d"}, {Value: "e"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	// 视口仅 2 行（height 7 = 标题/分隔/列表2/分隔/输入/状态），列表 5 项超出视口
	nm, _ := m.Update(tea.WindowSizeMsg{Height: 7, Width: 80})
	m = nm.(model)
	if m.cursor != 0 {
		t.Fatalf("initial cursor = %d, want 0", m.cursor)
	}
	// 首项按上键 → 回到末项
	m = pressKey(t, m, "up")
	if m.cursor != 4 {
		t.Fatalf("cursor after up from first = %d, want 4", m.cursor)
	}
	// 视口跟随跳到末尾：光标可见（scrollY 应为 4-2+1=3）
	if m.scrollY != 3 {
		t.Fatalf("scrollY = %d, want 3 (cursor visible at bottom)", m.scrollY)
	}
}

func TestCursorWrapsDownFromLast(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}, {Value: "c"}, {Value: "d"}, {Value: "e"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	nm, _ := m.Update(tea.WindowSizeMsg{Height: 7, Width: 80})
	m = nm.(model)
	m.cursor = 4
	m.scrollY = 3 // 模拟已滚动到底部
	// 末项按下键 → 回到首项，视口回到顶部
	m = pressKey(t, m, "down")
	if m.cursor != 0 {
		t.Fatalf("cursor after down from last = %d, want 0", m.cursor)
	}
	if m.scrollY != 0 {
		t.Fatalf("scrollY = %d, want 0 (scrolled back to top)", m.scrollY)
	}
	// 循环之后继续按下键 → 依次前进
	m = pressKey(t, m, "down")
	if m.cursor != 1 {
		t.Fatalf("cursor after second down = %d, want 1", m.cursor)
	}
}

func TestCursorWrapWithJK(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = pressKey(t, m, "k") // 首项按 k → 末项
	if m.cursor != 2 {
		t.Fatalf("cursor after k from first = %d, want 2", m.cursor)
	}
	m = pressKey(t, m, "j") // 末项按 j → 首项
	if m.cursor != 0 {
		t.Fatalf("cursor after j from last = %d, want 0", m.cursor)
	}
}

func TestCursorNoWrapOnEmptyOrSingle(t *testing.T) {
	// 空过滤结果：上下键不移动、不 panic
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = typeKeys(t, m, "zzz") // 过滤为空
	if len(m.filtered) != 0 {
		t.Fatalf("filtered = %v, want empty", m.filtered)
	}
	m = pressKey(t, m, "up")
	m = pressKey(t, m, "down")
	if m.cursor != 0 {
		t.Fatalf("cursor on empty list = %d, want 0", m.cursor)
	}

	// 单个候选：上下键原地不动
	m = newModel(Options{
		Cands:      []Candidate{{Value: "a"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = pressKey(t, m, "up")
	m = pressKey(t, m, "down")
	if m.cursor != 0 {
		t.Fatalf("cursor on single item = %d, want 0", m.cursor)
	}
}

// ---- 数字直选（JumpKeys）与取消 ----

func TestDigitJumpSelectsWhenJumpKeys(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}, {Value: "c"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
		JumpKeys:   true,
	}, 0)
	nm, _ := m.Update(keyMsg("2")) // 无输入时数字 2 → 直选第 2 项并退出
	m = nm.(model)
	if m.cursor != 1 {
		t.Fatalf("cursor after digit jump = %d, want 1", m.cursor)
	}
}

func TestDigitTypedAsFilterWithoutJumpKeys(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a1"}, {Value: "x1"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = typeKeys(t, m, "1") // 无 JumpKeys：数字作为过滤字符（不触发直选退出）
	if m.query != "1" || len(m.filtered) != 2 {
		t.Fatalf("query=%q filtered=%v, want query \"1\" and both candidates", m.query, m.filtered)
	}
}

func TestDigitJumpDisabledAfterTyping(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a1"}, {Value: "b2"}, {Value: "c3"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
		JumpKeys:   true,
	}, 0)
	m = typeKeys(t, m, "b") // 已输入过滤词
	nm, _ := m.Update(keyMsg("2"))
	m = nm.(model)
	// 输入后数字参与过滤：query "b2"，不直选
	if m.query != "b2" {
		t.Fatalf("query = %q, want b2", m.query)
	}
	if len(m.filtered) != 1 || m.filtered[0].Value != "b2" {
		t.Fatalf("filtered = %v, want [b2]", m.filtered)
	}
}

func TestEscCancels(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	nm, _ := m.Update(keyMsg("esc"))
	m = nm.(model)
	if !m.cancelled {
		t.Fatal("esc should mark cancelled")
	}
}

func TestBackspaceAndCtrlU(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "alpha"}, {Value: "beta"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = typeKeys(t, m, "al")
	if m.query != "al" {
		t.Fatalf("query = %q, want al", m.query)
	}
	nm, _ := m.Update(keyMsg("backspace"))
	m = nm.(model)
	if m.query != "a" {
		t.Fatalf("query after backspace = %q, want a", m.query)
	}
	nm, _ = m.Update(keyMsg("ctrl+u"))
	m = nm.(model)
	if m.query != "" {
		t.Fatalf("query after ctrl+u = %q, want empty", m.query)
	}
}

func TestCtrlWDeletesLastWord(t *testing.T) {
	m := newModel(Options{
		Cands:      []Candidate{{Value: "a"}, {Value: "b"}},
		FilterOpts: FilterOptions{Mode: MatchSubstring},
	}, 0)
	m = typeKeys(t, m, "ele dev")
	nm, _ := m.Update(keyMsg("ctrl+w"))
	m = nm.(model)
	if m.query != "ele" {
		t.Fatalf("query after ctrl+w = %q, want ele", m.query)
	}
}
