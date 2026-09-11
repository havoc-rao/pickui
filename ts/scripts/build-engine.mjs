// pretest：为集成测试构建（或定位）picktui 引擎二进制。
//
// 顺序：$PICKTUI_BIN → 本目录缓存 .engine-bin/picktui → go build（引擎 go.mod
// 位于 go/ 子目录，包 ./cmd/picktui）→ 失败时报错（提示先安装 Go 或构建引擎）。
import { execFileSync } from 'node:child_process';
import { existsSync, mkdirSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const pkgRoot = join(here, '..');
const repoRoot = join(pkgRoot, '..', 'go');
const binName = process.platform === 'win32' ? 'picktui.exe' : 'picktui';
const outFile = join(pkgRoot, '.engine-bin', binName);

if (process.env.PICKTUI_BIN) {
  console.log(`[build-engine] using PICKTUI_BIN=${process.env.PICKTUI_BIN}`);
  process.exit(0);
}
if (existsSync(outFile)) {
  console.log(`[build-engine] cached ${outFile}`);
  process.exit(0);
}
mkdirSync(dirname(outFile), { recursive: true });
try {
  execFileSync('go', ['build', '-o', outFile, './cmd/picktui'], {
    cwd: repoRoot,
    stdio: 'inherit',
    env: { ...process.env, GOCACHE: join(pkgRoot, '.engine-bin', '.gocache') },
  });
  console.log(`[build-engine] built ${outFile}`);
} catch {
  console.error(
    '[build-engine] 无法构建引擎二进制：需要 Go 工具链（或设置 PICKTUI_BIN 指向已构建的 picktui）。',
  );
  process.exit(1);
}