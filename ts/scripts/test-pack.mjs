// test-pack — 打包为 npm 安装包并在干净目录验证「作为包直接调用」：
//   import { filter } from '@havocrao/pickui'
//   import { pick } from '@havocrao/pickui/tui'
//   const { histGet } = require('@havocrao/pickui')
//
// 用法：npm run test:pack（依赖已构建的 dist 与引擎二进制）。
import { execFileSync } from 'node:child_process';
import { mkdtempSync, rmSync, existsSync, mkdirSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pkgRoot = join(here, '..');
const engineCandidates = [
  process.env.PICKUI_BIN,
  join(pkgRoot, '.engine-bin', process.platform === 'win32' ? 'pickui.exe' : 'pickui'),
  join(pkgRoot, '..', 'dist', process.platform === 'win32' ? 'pickui.exe' : 'pickui'),
].filter(Boolean);
const engine = engineCandidates.find((c) => existsSync(c));
if (!engine) {
  console.error('[test-pack] 引擎二进制缺失：先运行 npm run pretest 或设置 PICKUI_BIN');
  process.exit(1);
}

const work = mkdtempSync(join(tmpdir(), 'pickui-pack-'));
const npmEnv = { ...process.env, npm_config_cache: join(work, 'npm-cache') };
try {
  // npm pack 产物名：<pkg-name>-<version>.tgz（--silent 时 stdout 即文件名）
  const packOut = execFileSync('npm', ['pack', '--silent', '--pack-destination', work], {
    cwd: pkgRoot,
    env: npmEnv,
    encoding: 'utf8',
    stdio: ['ignore', 'pipe', 'inherit'],
  });
  const tarball = join(work, packOut.trim().split('\n').pop() ?? '');
  if (!existsSync(tarball)) {
    throw new Error(`npm pack 产物未找到：${tarball}`);
  }

  // 安装到干净目录（无运行时依赖，无需网络）
  mkdirSync(join(work, 'app'), { recursive: true });
  execFileSync('npm', ['install', '--no-audit', '--no-fund', '--silent', tarball], {
    cwd: join(work, 'app'),
    env: npmEnv,
    stdio: 'inherit',
  });

  const env = { ...process.env, PICKUI_BIN: engine };
  const smoke = `
    const assert = require('node:assert/strict');
    (async () => {
      // 隔离数据目录（hist/confirm 状态不落真实配置）
      process.env.PICKUI_CONFIG_DIR =
        require('node:fs').mkdtempSync(require('node:os').tmpdir() + '/pickui-pack-state-');
      // ESM：聚合入口 + 子路径（tree-shakable 契约）
      const { filter, resolve, histGet, histSet, confirmAdd, confirmCheck, engineVersion } =
        await import('@havocrao/pickui');
      const { pick } = await import('@havocrao/pickui/tui');
      const { menu } = await import('@havocrao/pickui/tui');
      // CJS：require 聚合入口
      const cjs = require('@havocrao/pickui');

      const hits = await filter(['electron:dev', 'electron:build'], 'ele dev');
      assert.deepEqual(hits.map(h => h.value), ['electron:dev']);
      assert.equal(await resolve(['build', 'release', 'serve'], 're'), 'release');
      assert.equal(await resolve(['release', 'restart'], 're'), null);

      process.env.PICKUI_PICK = 'off';
      assert.equal(await pick(['alpha', 'beta'], { query: 'beta' }), 'beta');

      await histSet('pack test', 'value');
      assert.equal(await histGet('pack test'), 'value');
      assert.equal(await confirmAdd('pack test', 'x'), true);
      assert.equal(await confirmCheck('pack test', 'x'), true);
      assert.equal(typeof cjs.pick, 'function');
      assert.match(await engineVersion(), /^\\d+\\.\\d+\\.\\d+$/);

      await menu('only-label', []).then(
        () => { throw new Error('menu usage 应拒绝'); },
        (e) => assert.equal(e.exitCode, 2),
      );
      console.log('[test-pack] OK: esm + subpath + cjs 全部可调用');
    })().catch((e) => { console.error(e); process.exit(1); });
  `;

  execFileSync(process.execPath, ['-e', smoke], { cwd: join(work, 'app'), env, stdio: 'inherit' });
} finally {
  rmSync(work, { recursive: true, force: true });
}