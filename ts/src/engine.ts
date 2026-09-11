/**
 * engine — 引擎发现与进程调用（薄绑定核心，零渲染逻辑）。
 *
 * 引擎发现顺序（对齐 docs/integration/protocol.md）：
 *   1. $PICKTUI_BIN
 *   2. $PATH 中的 picktui / picktui.exe
 *
 * 渲染/按键/匹配全部在引擎内完成；本模块只负责 spawn + stdout/stderr 收集，
 * 保证宿主 stdout 不被污染（交互 TUI 走引擎的 /dev/tty）。
 */
import { spawn } from 'node:child_process';
import { existsSync } from 'node:fs';
import { delimiter, join } from 'node:path';

import { PicktuiError } from './types.js';

/** 绑定声明的最低引擎版本（协议 v1）。 */
export const MIN_ENGINE_VERSION = '0.1.0';

/** 引擎二进制名（win32 带 .exe 后缀）。 */
function binName(): string {
  return process.platform === 'win32' ? 'picktui.exe' : 'picktui';
}

/** 解析引擎二进制绝对路径；找不到时抛出带指引的错误。 */
export async function engineBin(): Promise<string> {
  const fromEnv = process.env.PICKTUI_BIN;
  if (fromEnv) {
    return fromEnv;
  }
  const name = binName();
  const dirs = (process.env.PATH ?? '').split(delimiter).filter(Boolean);
  for (const dir of dirs) {
    const candidate = join(dir, name);
    if (existsSync(candidate)) {
      return candidate;
    }
  }
  throw new Error(
    `picktui 引擎未找到：请设置 PICKTUI_BIN 指向引擎二进制，或将 ${name} 加入 PATH。` +
      `（引擎即本仓库 Go 模块的 cmd/picktui，` +
      `见 README「引擎二进制」一节；` +
      `纯数据场景可设 PICKTUI_NO_BIN=1 使用内嵌 JS 过滤实现——该实现尚未提供，请先安装引擎。）`,
  );
}

/** 引擎调用的原始结果。 */
export interface EngineInvocation {
  exitCode: number;
  stdout: string;
  stderr: string;
}

/** 调用引擎子命令：数据走 stdin，结果回 stdout/stderr（无 TTY 依赖）。 */
export async function invokeEngine(
  args: string[],
  opts: { input?: string; env?: NodeJS.ProcessEnv } = {},
): Promise<EngineInvocation> {
  const bin = await engineBin();
  return new Promise((resolve, reject) => {
    let settled = false;
    const fail = (err: unknown) => {
      if (!settled) {
        settled = true;
        reject(err);
      }
    };
    const done = (inv: EngineInvocation) => {
      if (!settled) {
        settled = true;
        resolve(inv);
      }
    };

    const child = spawn(bin, args, {
      stdio: ['pipe', 'pipe', 'pipe'],
      env: { ...process.env, ...opts.env },
      // 引擎自身通过 /dev/tty 完成交互渲染与输入；这里不需要终端
      windowsHide: true,
    });

    let stdout = '';
    let stderr = '';
    child.stdout.setEncoding('utf8');
    child.stderr.setEncoding('utf8');
    child.stdout.on('data', (chunk: string) => (stdout += chunk));
    child.stderr.on('data', (chunk: string) => (stderr += chunk));
    child.on('error', fail);
    child.on('close', (code) => done({ exitCode: code ?? -1, stdout, stderr }));

    if (opts.input !== undefined) {
      child.stdin.end(opts.input);
    } else {
      child.stdin.end();
    }
  });
}

/** 引擎调用约定：退出码 0 成功 / 1 运行错误 / 2 用法错误 / 130 取消。 */
export function assertSuccess(
  inv: EngineInvocation,
  what: string,
): void {
  if (inv.exitCode === 0) {
    return;
  }
  const detail = inv.stderr.trim() || `exit code ${inv.exitCode}`;
  throw new PicktuiError(`${what}失败：${detail}`, inv.exitCode, inv.stderr);
}

/** 解析 stdout 为 JSON（协议输出恒为 JSON）；失败抛 PicktuiError。 */
export function parseJSON<T>(stdout: string, what: string): T {
  try {
    return JSON.parse(stdout) as T;
  } catch {
    throw new PicktuiError(`${what}输出不是合法 JSON：${stdout.slice(0, 200)}`, -1, stdout);
  }
}

/** 读取引擎版本（"picktui <semver>"）。 */
export async function engineVersion(): Promise<string> {
  const bin = await engineBin();
  const inv = await invokeEngine(['version']);
  assertSuccess(inv, 'picktui version');
  const match = /^picktui\s+(\d+\.\d+\.\d+)/.exec(inv.stdout.trim());
  if (!match) {
    throw new PicktuiError(`无法解析引擎版本：${inv.stdout.trim()}`, inv.exitCode, inv.stdout);
  }
  return match[1];
}

/** 比较两个 x.y.z 版本号：a >= b。 */
export function versionAtLeast(a: string, b: string): boolean {
  const pa = a.split('.').map(Number);
  const pb = b.split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    const x = pa[i] ?? 0;
    const y = pb[i] ?? 0;
    if (x !== y) {
      return x > y;
    }
  }
  return true;
}

/** 校验引擎版本不低于绑定声明的最低版本；不满足时抛错。 */
export async function assertEngineVersion(min = MIN_ENGINE_VERSION): Promise<string> {
  const v = await engineVersion();
  if (!versionAtLeast(v, min)) {
    throw new PicktuiError(
      `picktui 引擎版本 ${v} 低于绑定要求的最低版本 ${min}（请升级引擎或设置 PICKTUI_BIN）`,
      -1,
      '',
    );
  }
  return v;
}

/** 将候选列表序列化为协议结构化行（key<TAB>description）。 */
export function serializeCandidates(cands: (string | import('./types.js').Candidate)[]): string {
  return cands
    .map((c) => (typeof c === 'string' ? c : c.desc ? `${c.value}\t${c.desc}` : c.value))
    .join('\n') + '\n';
}