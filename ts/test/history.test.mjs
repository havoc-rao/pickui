// history / confirm — 文件状态（纯 TS，TOML 格式与 Go 引擎兼容）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { picktui, withConfigDir } from './helper.mjs';

const { histGet, histSet, confirmCheck, confirmAdd } = picktui;

test('hist set → get 往返，无记录返回空串', async () => {
  const dir = withConfigDir(test);
  await histSet('git p', 'pull');
  assert.equal(await histGet('git p'), 'pull');
  assert.equal(await histGet('never set'), '');
  assert.ok(existsSync(join(dir, 'history.toml')));
});

test('hist 覆盖写回，多 label 互不干扰', async () => {
  withConfigDir(test);
  await histSet('git co', 'main');
  await histSet('git co', 'develop');
  await histSet('other', 'x');
  assert.equal(await histGet('git co'), 'develop');
  assert.equal(await histGet('other'), 'x');
});

test('history.toml 文件格式与 Go 引擎兼容（可被引擎 re-read）', async () => {
  const dir = withConfigDir(test);
  await histSet('git p', 'pull');
  await histSet('git co', 'feature/x');
  const text = readFileSync(join(dir, 'history.toml'), 'utf8');
  // 表头注释 + 双引号 key（含空格）+ last 赋值，与 BurntSushi/toml 输出同构
  assert.match(text, /^# picktui history — last selection per key \(auto-managed\)/);
  assert.match(text, /\["git p"\]/);
  assert.match(text, /last = "pull"/);
  assert.match(text, /\["git co"\]/);
  assert.match(text, /last = "feature\/x"/);
});

test('hist set 失败（数据目录被文件占用）→ PicktuiError', async () => {
  const { mkdtempSync } = await import('node:fs');
  const { tmpdir } = await import('node:os');
  const dir = mkdtempSync(join(tmpdir(), 'picktui-blocked-'));
  const blocked = join(dir, 'not-a-dir');
  writeFileSync(blocked, 'x');
  const prev = process.env.PICKTUI_CONFIG_DIR;
  process.env.PICKTUI_CONFIG_DIR = blocked;
  try {
    await assert.rejects(
      () => histSet('git p', 'pull'),
      (err) => {
        assert.equal(err.name, 'PicktuiError');
        assert.equal(err.exitCode, 1);
        return true;
      },
    );
  } finally {
    if (prev === undefined) {
      delete process.env.PICKTUI_CONFIG_DIR;
    } else {
      process.env.PICKTUI_CONFIG_DIR = prev;
    }
  }
});

test('legacy picks.toml 自动迁移到 history.toml', async () => {
  const dir = withConfigDir(test);
  writeFileSync(
    join(dir, 'picks.toml'),
    '# old\n["git p"]\nlast = "pull"\n',
  );
  assert.equal(await histGet('git p'), 'pull');
  assert.ok(!existsSync(join(dir, 'picks.toml')), '旧文件应被删除');
  assert.ok(existsSync(join(dir, 'history.toml')));
});

test('confirm check/add 往返（幂等、跨 label 隔离）', async () => {
  const dir = withConfigDir(test);
  assert.equal(await confirmCheck('npm run', 'release'), false);
  assert.equal(await confirmAdd('npm run', 'release'), true);
  assert.equal(await confirmCheck('npm run', 'release'), true);
  assert.equal(await confirmCheck('npm run', 'dev'), false);
  assert.equal(await confirmCheck('docker exec', 'release'), false);
  assert.ok(existsSync(join(dir, 'confirm.toml')));
});

test('confirm.toml 文件格式与 Go 引擎兼容', async () => {
  const dir = withConfigDir(test);
  await confirmAdd('npm run', 'release');
  await confirmAdd('npm run', 'dev');
  const text = readFileSync(join(dir, 'confirm.toml'), 'utf8');
  assert.match(text, /^# picktui confirm — user-confirmed auto resolutions \(auto-managed\)/);
  assert.match(text, /\[confirmed\]/);
  assert.match(text, /"npm run" = \["release", "dev"\]/);
});

test('confirm add 幂等：重复添加不产生重复记录', async () => {
  withConfigDir(test);
  await confirmAdd('git co', 'main');
  await confirmAdd('git co', 'main');
  assert.equal(await confirmCheck('git co', 'main'), true);
});

test('confirm 读取 Go 引擎写出的文件（外部格式兼容）', async () => {
  const dir = withConfigDir(test);
  mkdirSync(dir, { recursive: true });
  writeFileSync(
    join(dir, 'confirm.toml'),
    '# picktui confirm — user-confirmed auto resolutions (auto-managed)\n' +
      '[confirmed]\n' +
      '"docker exec" = ["release", "dev"]\n',
  );
  assert.equal(await confirmCheck('docker exec', 'release'), true);
  assert.equal(await confirmCheck('docker exec', 'other'), false);
});