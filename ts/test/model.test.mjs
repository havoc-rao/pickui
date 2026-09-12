// model — TUI 状态机（纯逻辑，对齐 Go 引擎 model 按键行为与渲染布局）。
import test from 'node:test';
import assert from 'node:assert/strict';
import {
  newModel,
  update,
  view,
  truncateWidth,
  charWidth,
  setColorEnabled,
} from '../dist/esm/model.js';

const cands = ['dev', 'build', 'release', 'serve', 'test', 'lint'].map((value) => ({
  value,
  desc: '',
}));

function make(opts = {}) {
  return newModel({
    title: 'picktui pick',
    subtitle: '',
    cands,
    query: '',
    filterOpts: { mode: 'substring' },
    initial: 0,
    jumpKeys: false,
    statusHint: 'hint',
    multi: false,
    selected: [],
    ...opts,
  });
}

test('初始：全部候选、光标 0、未退出', () => {
  const m = make();
  assert.equal(m.filtered.length, 6);
  assert.equal(m.cursor, 0);
  assert.equal(m.query, '');
  assert.equal(m.quit, false);
});

test('输入过滤：append query 并重算', () => {
  let m = make();
  m = update(m, 'd');
  assert.equal(m.query, 'd');
  // 'd' 命中 dev 与 build（build 含字母 d）
  assert.deepEqual(m.filtered.map((c) => c.value), ['dev', 'build']);
});

test('backspace / ctrl+u / ctrl+w 编辑', () => {
  let m = make();
  for (const ch of ['r', 'e', 'l', 'e', 'a', 's', 'e']) {
    m = update(m, ch);
  }
  m = update(m, 'backspace');
  assert.equal(m.query, 'releas');
  m = update(m, 'ctrl+u');
  assert.equal(m.query, '');
  assert.equal(m.filtered.length, 6);

  m = make();
  for (const ch of ['e', 'l', ' ', 'l', 'i']) {
    m = update(m, ch);
  }
  m = update(m, 'ctrl+w');
  assert.equal(m.query, 'el'); // 去掉最后一词 "li"
});

test('循环移动：up/down 越界回绕', () => {
  let m = make();
  m = update(m, 'up');
  assert.equal(m.cursor, 5); // 首项按上键到末项
  m = update(m, 'down');
  assert.equal(m.cursor, 0);
  m = update(m, 'j'); // j = down
  assert.equal(m.cursor, 1);
  m = update(m, 'k'); // k = up
  assert.equal(m.cursor, 0);
  m = update(m, 'ctrl+n');
  assert.equal(m.cursor, 1);
  m = update(m, 'ctrl+p');
  assert.equal(m.cursor, 0);
  m = update(m, 'tab');
  assert.equal(m.cursor, 1);
  m = update(m, 'up'); // 1 → 0
  assert.equal(m.cursor, 0);
  m = update(m, 'up'); // 0 → 末项回绕
  assert.equal(m.cursor, 5);
});

test('enter 选中（filtered 非空）；esc/ctrl+c 取消', () => {
  let m = make();
  m = update(update(m, 'down'), 'enter');
  assert.equal(m.quit, true);
  assert.equal(m.cancelled, false);
  assert.equal(m.filtered[m.cursor].value, 'build');

  m = make();
  m = update(m, 'esc');
  assert.equal(m.cancelled, true);
  assert.equal(m.quit, true);

  m = make();
  m = update(m, 'ctrl+c');
  assert.equal(m.cancelled, true);

  // 过滤结果为空时 enter 不退出
  m = make();
  m = update(m, 'zzz');
  assert.equal(m.filtered.length, 0);
  m = update(m, 'enter');
  assert.equal(m.quit, false);
});

test('过滤后光标 clamp 在有效范围', () => {
  let m = make();
  m = update(m, 'down'); // cursor = 1
  m = update(m, 're'); // 过滤只剩 release（唯一命中）→ cursor clamp 0
  assert.equal(m.filtered.length, 1);
  assert.equal(m.cursor, 0);

  m = make();
  m = update(m, 'zzz'); // 无命中 → filtered 空、cursor 0
  assert.equal(m.filtered.length, 0);
  assert.equal(m.cursor, 0);
});

test('jumpKeys 数字直选（无输入时）', () => {
  let m = make({ jumpKeys: true });
  m = update(m, '3'); // 直选第 3 项
  assert.equal(m.cursor, 2);
  assert.equal(m.quit, true);
  // 有输入时数字作为过滤字符
  m = make({ jumpKeys: true });
  m = update(update(m, 'b'), '2');
  assert.equal(m.query, 'b2');
  assert.equal(m.quit, false);
});

test('space：单选模式是过滤字符，多选模式勾选/取消', () => {
  let m = make();
  m = update(m, 'space');
  assert.equal(m.query, ' ');
  assert.equal(m.filtered.length, 6); // 纯空格 = 空查询（全部保留）

  m = make({ multi: true });
  m = update(m, 'space');
  assert.ok(m.selected.has('dev'));
  m = update(m, 'down');
  m = update(m, 'space');
  assert.ok(m.selected.has('build'));
  m = update(m, 'up');
  m = update(m, 'space');
  assert.ok(!m.selected.has('dev'), '再次空格取消勾选');
});

test('view：布局（NO_COLOR 关闭时纯文本）', () => {
  setColorEnabled(false);
  try {
    let m = make();
    m = update(m, 'de');
    const frame = view(m);
    const lines = frame.trimEnd().split('\n');
    assert.match(lines[0], /picktui pick/); // 标题
    assert.equal(lines[1], '─'.repeat(80)); // 分隔线（默认宽 80）
    assert.match(lines[2], /▸ dev/); // 光标行
    assert.equal(lines[lines.length - 3], '─'.repeat(80)); // 底部分隔线
    assert.match(lines[lines.length - 2], /❯ de/); // 输入行
    assert.match(lines[lines.length - 1], /1\/6/); // 计数
    assert.match(lines[lines.length - 1], /hint/); // 状态提示
  } finally {
    setColorEnabled(true);
  }
});

test('view：命中高亮 ANSI（粉色 213）只出现在非光标行', () => {
  setColorEnabled(true);
  try {
    let m = make();
    m = update(m, 'de');
    const frame = view(m);
    const lines = frame.trimEnd().split('\n');
    // 光标行整行绿粗体（120），不嵌套命中高亮
    assert.match(lines[2], /38;5;120m/);
    assert.ok(!lines[2].includes('38;5;213m'), '光标行不应有命中粉高亮');
  } finally {
    setColorEnabled(false);
  }
});

test('view：描述截断与显示宽度', () => {
  setColorEnabled(false);
  try {
    const withDesc = (value, desc) => ({ value, desc });
    const m = newModel({
      title: 't',
      subtitle: '',
      cands: [withDesc('dev', '短描述')],
      query: '',
      filterOpts: { mode: 'substring' },
      initial: 0,
      jumpKeys: false,
      statusHint: '',
      multi: false,
      selected: [],
      width: 30,
    });
    const frame = view(m);
    assert.match(frame, /dev  短描述/);
  } finally {
    setColorEnabled(true);
  }
});

test('truncateWidth / charWidth（CJK 宽 2、emoji 宽 2、ASCII 1）', () => {
  assert.equal(charWidth('a'), 1);
  assert.equal(charWidth('中'), 2);
  assert.equal(charWidth('🎉'), 2);
  assert.equal(truncateWidth('hello world', 8), 'hello w…');
  assert.equal(truncateWidth('abc', 10), 'abc');
  assert.equal(truncateWidth('中文很长', 6), '中文…');
});

test('select1 语义由 pick 流程处理，model 只保证 filtered 顺序', () => {
  const m = make();
  assert.deepEqual(
    m.filtered.map((c) => c.value),
    ['dev', 'build', 'release', 'serve', 'test', 'lint'],
  );
});