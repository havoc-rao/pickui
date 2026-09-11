// helper — 测试公共设施：定位引擎二进制、隔离数据目录、导入构建产物。
import { existsSync, mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pkgRoot = join(here, '..');

/** 引擎发现：$PICKTUI_BIN → .engine-bin 缓存 → 仓库 dist（与引擎侧顺序一致）。 */
export function locateEngine() {
  const name = process.platform === 'win32' ? 'picktui.exe' : 'picktui';
  const candidates = [
    process.env.PICKTUI_BIN,
    join(pkgRoot, '.engine-bin', name),
    join(pkgRoot, '..', 'dist', name),
  ].filter(Boolean);
  for (const c of candidates) {
    if (existsSync(c)) {
      return c;
    }
  }
  throw new Error(
    '引擎二进制缺失：请先运行 npm run pretest（需要 Go 工具链）或设置 PICKTUI_BIN',
  );
}

// 必须在 import 包之前确定引擎位置
process.env.PICKTUI_BIN = locateEngine();

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