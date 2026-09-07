// tui — pick 非交互路径 / menu 用法错误（交互 TUI 需要真实 /dev/tty，
// 由 Go 侧 model 测试覆盖；本文件全部确定性执行）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { pickui, withConfigDir, withEnv } from './helper.mjs';

const { pick, menu, rawPick, histGet } = pickui;

test('pick 非交互（PICKUI_PICK=off）：无 query 取首个', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  assert.equal(await pick(['alpha', 'beta', 'gamma']), 'alpha');
});

test('pick 非交互：query 过滤取首个匹配', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  assert.equal(await pick(['alpha', 'beta', 'gamma'], { query: 'beta' }), 'beta');
  // 无匹配 → 引擎退出码 1 → PickuiError（绝不静默回退首个）
  await assert.rejects(
    () => pick(['alpha', 'beta'], { query: 'zzz' }),
    (err) => {
      assert.equal(err.name, 'PickuiError');
      assert.equal(err.exitCode, 1);
      assert.match(err.stderr, /no match for "zzz"/);
      return true;
    },
  );
});

test('pick --label 记录选择记忆（与 histGet 同源）', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  withConfigDir(test);
  assert.equal(await pick(['build', 'dev', 'serve'], { query: 'dev', label: 'npm run' }), 'dev');
  assert.equal(await histGet('npm run'), 'dev');
});

test('pick -1 唯一匹配自动选中', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  assert.equal(await pick(['uniq', 'alpha'], { query: 'uniq', select1: true }), 'uniq');
});

test('rawPick 透传引擎 flags（--from 候选来源）', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  assert.equal(
    await rawPick(['--from', 'printf "main\\nfeature/x\\n"', '-q', 'feat']),
    'feature/x',
  );
});

test('menu 参数不足 → PickuiError（用法错误退出码 2）', async () => {
  withEnv(test, 'PICKUI_PICK', 'off');
  await assert.rejects(
    () => menu('only-label', []),
    (err) => {
      assert.equal(err.name, 'PickuiError');
      assert.equal(err.exitCode, 2);
      assert.match(err.stderr, /usage: pickui _menu/);
      return true;
    },
  );
});