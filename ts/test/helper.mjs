// helper — 测试公共设施：导入构建产物、隔离数据目录（纯 TS 实现，无引擎）。
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

/** 构建产物（ESM）。 */
export const picktui = await import('../dist/esm/index.js');

/** 隔离数据目录（hist/confirm 状态），测试后自动还原。 */
export function withConfigDir(t) {
  const dir = mkdtempSync(join(tmpdir(), 'picktui-ts-'));
  const prev = process.env.PICKTUI_CONFIG_DIR;
  process.env.PICKTUI_CONFIG_DIR = dir;
  t.after(() => {
    if (prev === undefined) {
      delete process.env.PICKTUI_CONFIG_DIR;
    } else {
      process.env.PICKTUI_CONFIG_DIR = prev;
    }
  });
  return dir;
}

/** 临时设置并还原单个环境变量。 */
export function withEnv(t, key, value) {
  const prev = process.env[key];
  process.env[key] = value;
  t.after(() => {
    if (prev === undefined) {
      delete process.env[key];
    } else {
      process.env[key] = prev;
    }
  });
}