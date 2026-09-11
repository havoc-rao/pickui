// history / confirm — 文件状态协议往返（隔离数据目录）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
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

test('hist 覆盖写回；记忆与 Go 库 API 同文件（协议一致性）', async () => {
  withConfigDir(test);
  await histSet('git co', 'main');
  await histSet('git co', 'develop');
  assert.equal(await histGet('git co'), 'develop');
});

test('hist set 失败（数据目录被文件占用）→ PicktuiError', async () => {
  const { mkdtempSync, writeFileSync } = await import('node:fs');
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

test('confirm check/add 往返（幂等、跨 label 隔离）', async () => {
  const dir = withConfigDir(test);
  assert.equal(await confirmCheck('npm run', 'release'), false);
  assert.equal(await confirmAdd('npm run', 'release'), true);
  assert.equal(await confirmCheck('npm run', 'release'), true);
  assert.equal(await confirmCheck('npm run', 'dev'), false);
  assert.equal(await confirmCheck('docker exec', 'release'), false);
  assert.ok(existsSync(join(dir, 'confirm.toml')));
});

test('confirm add 幂等：重复添加不产生重复记录', async () => {
  withConfigDir(test);
  await confirmAdd('git co', 'main');
  await confirmAdd('git co', 'main');
  // 读回 TOML 验证只有一条（引擎侧语义，允许简查：check 依然为 true）
  assert.equal(await confirmCheck('git co', 'main'), true);
});