/**
 * model — 过滤选择器 TUI 状态机（纯逻辑，无 IO，可独立单测）。
 *
 * 对齐 Go 引擎 model：布局为 标题栏 / 分隔线 / 列表（视口滚动）/ 分隔线 /
 * 输入行（闪烁光标）/ 状态栏；按键行为（循环移动、数字直选、多选勾选、
 * ctrl+u/w 编辑）与引擎逐键一致。
 *
 * 本模块不接触终端：交互循环（tty.ts）只负责 按键 → update → view → 渲染。
 */
import { filterCands, highlightRanges } from './filter.js';
import type { Candidate, FilterOptions } from './types.js';

/** 气泡风格配色开关（NO_COLOR 环境变量时降级纯文本，保障脚本输出干净）。 */
let colorEnabled = true;
export function setColorEnabled(v: boolean): void {
  colorEnabled = v;
}
export function isColorEnabled(): boolean {
  return colorEnabled;
}

/** 样式（对齐引擎 lipgloss 调色：120 亮绿 / 241 灰 / 36 青绿 / 213 粉 / 237 分隔）。 */
const STYLES = {
  title: '\x1b[1;38;5;120m',
  dim: '\x1b[38;5;241m',
  ok: '\x1b[1;38;5;36m',
  hl: '\x1b[38;5;213m',
  cur: '\x1b[1;38;5;120m',
  sep: '\x1b[38;5;237m',
  prompt: '\x1b[1;38;5;120m',
  reset: '\x1b[0m',
};

/** 包裹样式（颜色关闭时原样返回）。 */
function paint(name: keyof typeof STYLES, s: string): string {
  if (!colorEnabled) {
    return s;
  }
  return STYLES[name] + s + STYLES.reset;
}

/** 过滤选择器配置。 */
export interface ModelOpts {
  title: string;
  subtitle?: string;
  cands: Candidate[];
  query: string;
  filterOpts: FilterOptions;
  initial: number;
  jumpKeys: boolean;
  statusHint: string;
  multi: boolean;
  selected: string[];
}

/** TUI 状态（不可变更新）。 */
export interface UiModel {
  opts: ModelOpts;
  filtered: Candidate[];
  query: string;
  cursor: number;
  scrollY: number;
  height: number;
  width: number;
  phase: number;
  cancelled: boolean;
  quit: boolean;
  selected: Set<string>;
}

/** 固定行数：标题(1) + 分隔线(1) + [列表] + 分隔线(1) + 输入(1) + 状态(1)。 */
const FIXED_LINES = 5;

/** 按键（数字 '1'...'9'、字母等作为字符串传入）。 */
export type KeyLike = string | { type: 'char'; value: string };

export function newModel(opts: ModelOpts): UiModel {
  let initial = opts.initial;
  if (initial < 0 || initial >= opts.cands.length) {
    initial = 0;
  }
  const m: UiModel = {
    opts,
    filtered: [],
    query: opts.query,
    cursor: initial,
    scrollY: 0,
    height: 0,
    width: 0,
    phase: 0,
    cancelled: false,
    quit: false,
    selected: new Set(opts.selected),
  };
  recompute(m);
  return m;
}

/** 多选模式：按 Cands 顺序返回勾选的 Value 列表。 */
export function selectedValues(m: UiModel): string[] {
  if (m.selected.size === 0) {
    return [];
  }
  const out: string[] = [];
  for (const c of m.opts.cands) {
    if (m.selected.has(c.value)) {
      out.push(c.value);
    }
  }
  return out;
}

/** 根据当前 query 重新过滤，保持光标在有效范围内。 */
function recompute(m: UiModel): void {
  m.filtered = filterCands(m.opts.cands, m.query, m.opts.filterOpts);
  if (m.cursor >= m.filtered.length) {
    m.cursor = m.filtered.length - 1;
  }
  if (m.cursor < 0) {
    m.cursor = 0;
  }
  ensureVisible(m);
}

/** 处理一个按键，返回新模型（不可变；变化时替换字段）。 */
export function update(m: UiModel, key: KeyLike): UiModel {
  switch (key) {
    case 'ctrl+c':
    case 'esc':
      return { ...m, cancelled: true, quit: true };
    case 'enter':
      if (m.filtered.length > 0) {
        return { ...m, quit: true };
      }
      return m;
    case 'space':
      // 多选模式：无过滤输入时空格勾选/取消；否则空格作为过滤字符
      if (m.opts.multi && m.query === '' && m.cursor < m.filtered.length) {
        const v = m.filtered[m.cursor].value;
        const next = new Set(m.selected);
        if (next.has(v)) {
          next.delete(v);
        } else {
          next.add(v);
        }
        return { ...m, selected: next };
      }
      return appendQuery(m, ' ');
    case 'up':
    case 'k':
    case 'ctrl+p':
      return stepCursor(m, -1);
    case 'down':
    case 'j':
    case 'ctrl+n':
    case 'tab':
      return stepCursor(m, 1);
    case 'backspace':
      if (m.query.length > 0) {
        const runes = [...m.query];
        runes.pop();
        return recomputeQuery(m, runes.join(''));
      }
      return m;
    case 'ctrl+u':
      return recomputeQuery(m, '');
    case 'ctrl+w':
      return recomputeQuery(m, dropLastField(m.query));
    default: {
      // 统一为字符串处理（tty 键 'k'/'j' 等已在上方 case 匹配；数字是字符串）
      const s = typeof key === 'object' ? key.value : key;
      // 数字直选（菜单）：无输入时 1-9 跳过逐行移动快速选中
      if (m.opts.jumpKeys && m.query === '' && /^[1-9]$/.test(s)) {
        const n = Number(s);
        if (n - 1 < m.filtered.length) {
          return { ...m, cursor: n - 1, quit: true };
        }
        return m;
      }
      if (s === '') {
        return m; // 被忽略的键（如左右方向键）
      }
      return appendQuery(m, s);
    }
  }
}

function appendQuery(m: UiModel, s: string): UiModel {
  return recomputeQuery(m, m.query + s);
}

function recomputeQuery(m: UiModel, q: string): UiModel {
  const next: UiModel = { ...m, query: q };
  recompute(next);
  return next;
}

/** 循环移动：首项按上键回到末项，末项按下键回到首项。 */
function stepCursor(m: UiModel, delta: number): UiModel {
  const n = m.filtered.length;
  if (n <= 1) {
    return m;
  }
  const cursor = (m.cursor + delta + n) % n;
  const next = { ...m, cursor };
  ensureVisible(next);
  return next;
}

/** ctrl+w：去掉最后一词（与 Go strings.Fields 同语义）。 */
function dropLastField(q: string): string {
  const fields = q.trim().split(/\s+/).filter(Boolean);
  if (fields.length === 0) {
    return '';
  }
  return fields.slice(0, -1).join(' ');
}

/** 调整视口使光标可见（固定行数 5）。 */
function ensureVisible(m: UiModel): void {
  const visibleRows = visibleRowsOf(m);
  if (m.cursor < m.scrollY) {
    m.scrollY = m.cursor;
  }
  if (m.cursor >= m.scrollY + visibleRows) {
    m.scrollY = m.cursor - visibleRows + 1;
  }
}

function visibleRowsOf(m: UiModel): number {
  const rows = m.height - FIXED_LINES;
  return rows > 0 ? rows : 10; // 尺寸未到达时的默认值
}

/** 渲染整屏画面（带 ANSI 样式）。 */
export function view(m: UiModel): string {
  const width = m.width > 0 ? m.width : 80;
  const sep = paint('sep', '─'.repeat(width));

  const out: string[] = [];
  // 标题栏
  let head = paint('title', m.opts.title);
  if (m.opts.subtitle) {
    head += paint('dim', '  ·  ' + m.opts.subtitle);
  }
  out.push(head, sep);

  // 列表（视口区间）
  const visibleRows = visibleRowsOf(m);
  const end = Math.min(m.scrollY + visibleRows, m.filtered.length);
  for (let i = m.scrollY; i < end; i++) {
    const c = m.filtered[i];
    let box = '';
    if (m.opts.multi) {
      box = m.selected.has(c.value) ? paint('ok', '[x] ') : '[ ] ';
    }
    if (i === m.cursor) {
      // 光标行：整行绿色粗体（不再嵌套命中高亮，避免内层 reset 破坏外层颜色）
      let line = '  ▸ ' + box + c.value;
      if (c.desc) {
        line += '  ' + c.desc;
      }
      const pad = width - displayWidth(line);
      if (pad > 0) {
        line += ' '.repeat(pad);
      }
      out.push(paint('cur', line));
    } else {
      out.push('    ' + box + renderCandidateLine(c, m.query, m.opts.filterOpts, width));
    }
  }
  if (m.filtered.length === 0) {
    out.push(paint('dim', '    (no matches)'));
  }

  // 分隔线 + 输入行 + 状态栏
  out.push(sep);
  const cursorBlock = m.phase === 1 ? ' ' : '▏';
  out.push(paint('prompt', '❯ ') + m.query + cursorBlock);
  const count = `${m.filtered.length}/${m.opts.cands.length}`;
  let status = '  ' + paint('ok', count);
  if (m.opts.statusHint) {
    status += paint('dim', '  ·  ' + m.opts.statusHint);
  }
  out.push(status);

  return out.join('\n') + '\n';
}

/** 渲染候选行：value 命中高亮 + 灰色描述（按显示宽度截断）。 */
function renderCandidateLine(
  c: Candidate,
  query: string,
  opts: FilterOptions,
  width: number,
): string {
  const value = renderHighlighted(c.value, query, opts);
  if (!c.desc) {
    return value;
  }
  // 描述仅占 value 之后的剩余宽度（减 2 个分隔空格），太窄就不展示
  const remaining = width - displayWidth(value) - 2;
  if (remaining < 4) {
    return value;
  }
  return value + '  ' + paint('dim', truncateWidth(c.desc, remaining));
}

/** 按显示宽度截断 s，超长末尾补 …。 */
export function truncateWidth(s: string, maxWidth: number): string {
  if (maxWidth <= 1) {
    return '…';
  }
  if (displayWidth(s) <= maxWidth) {
    return s;
  }
  let out = '';
  let w = 0;
  for (const r of s) {
    const rw = charWidth(r);
    if (w + rw > maxWidth - 1) {
      break;
    }
    out += r;
    w += rw;
  }
  return out + '…';
}

/** 渲染候选字符串，高亮匹配片段（粉色）。 */
function renderHighlighted(s: string, query: string, opts: FilterOptions): string {
  const ranges = highlightRanges(s, query, opts);
  if (ranges.length === 0) {
    return s;
  }
  const runes = [...s];
  let out = '';
  let pos = 0;
  for (const [start, end] of ranges) {
    if (pos < start) {
      out += runes.slice(pos, start).join('');
    }
    out += paint('hl', runes.slice(start, end).join(''));
    pos = end;
  }
  if (pos < runes.length) {
    out += runes.slice(pos).join('');
  }
  return out;
}

/** 字符显示宽度（简化 wcwidth：CJK/全角=2，控制符=0，其余=1）。 */
export function charWidth(ch: string): number {
  const cp = ch.codePointAt(0) ?? 0;
  if (cp < 0x20 || cp === 0x7f) {
    return 0;
  }
  if (isWide(cp)) {
    return 2;
  }
  return 1;
}

/** 字符串显示宽度。 */
export function displayWidth(s: string): number {
  let w = 0;
  for (const ch of s) {
    w += charWidth(ch);
  }
  return w;
}

/** East Asian Wide / Fullwidth / emoji 等宽字符区间判定。 */
function isWide(cp: number): boolean {
  return (
    (cp >= 0x1100 && cp <= 0x115f) || // Hangul Jamo
    (cp >= 0x2e80 && cp <= 0x303e) || // CJK Radicals 等
    (cp >= 0x3041 && cp <= 0x33ff) || // 假名 / CJK 符号
    (cp >= 0x3400 && cp <= 0x4dbf) || // CJK 扩展 A
    (cp >= 0x4e00 && cp <= 0x9fff) || // CJK 统一表意
    (cp >= 0xa000 && cp <= 0xa4cf) || // 彝文
    (cp >= 0xa960 && cp <= 0xa97f) || // Hangul Jamo Extended-A
    (cp >= 0xac00 && cp <= 0xd7a3) || // Hangul 音节
    (cp >= 0xf900 && cp <= 0xfaff) || // CJK 兼容表意
    (cp >= 0xfe10 && cp <= 0xfe19) || // 竖排形式
    (cp >= 0xfe30 && cp <= 0xfe52) ||
    (cp >= 0xfe54 && cp <= 0xfe66) ||
    (cp >= 0xfe68 && cp <= 0xfe6b) ||
    (cp >= 0xff00 && cp <= 0xff60) || // 全角形式
    (cp >= 0xffe0 && cp <= 0xffe6) ||
    (cp >= 0x1f1e6 && cp <= 0x1f1ff) || // 区域指示符（旗帜）
    (cp >= 0x1f300 && cp <= 0x1faff) // emoji
  );
}