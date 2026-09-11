/**
 * tui — 交互选择 / 菜单（引擎跑 TUI，宿主零渲染）。
 *
 * 交互渲染与按键输入走引擎的 /dev/tty；本模块的 stdout 仅收选中值——
 * 与 `$(picktui pick ...)` 同样的安全语义。无 TTY 时引擎自动退化
 * （无 query 取首个、有 query 过滤取首），绑定无需分支。
 */
import { assertSuccess, invokeEngine, serializeCandidates } from './engine.js';
import type { Candidate, PickFlags } from './types.js';

/**
 * pick — 交互过滤选择（fzf 风格）。
 *
 * @param cands 候选（经 stdin 传入，与位置参数等价且无 argv 长度限制）
 * @param flags 透传引擎 flags（query/label/sep/fuzzy/select1/auto）
 * @returns 选中值；用户取消（esc/ctrl+c，退出码 130）返回 null
 */
export async function pick(
  cands: (string | Candidate)[] = [],
  flags: PickFlags = {},
): Promise<string | null> {
  const args: string[] = ['pick'];
  if (flags.query !== undefined) {
    args.push('-q', flags.query);
  }
  if (flags.label !== undefined) {
    args.push('--label', flags.label);
  }
  if (flags.sep !== undefined) {
    args.push('--sep', flags.sep);
  }
  if (flags.fuzzy) {
    args.push('--fuzzy');
  }
  if (flags.select1) {
    args.push('-1');
  }
  if (flags.auto) {
    args.push('--auto');
  }
  const inv = await invokeEngine(args, { input: serializeCandidates(cands) });
  if (inv.exitCode === 130) {
    return null; // 取消
  }
  assertSuccess(inv, 'picktui pick');
  return inv.stdout.replace(/\r?\n$/, '');
}

/**
 * menu — 多值缩写 TUI 菜单（数字 1-9 直选）。
 *
 * @param label 菜单标题与选择记忆键
 * @param cands 候选（经位置参数传入，注意 argv 长度限制；超长场景用 pick）
 * @returns 选中值；取消返回 null
 */
export async function menu(label: string, cands: string[]): Promise<string | null> {
  const inv = await invokeEngine(['_menu', label, ...cands]);
  if (inv.exitCode === 130) {
    return null;
  }
  assertSuccess(inv, 'picktui _menu');
  return inv.stdout.replace(/\r?\n$/, '');
}

/**
 * rawPick — 透传任意引擎 pick 参数（--from/--map 等绑定未包装的 flags）。
 *
 * @example rawPick(['--from', 'git branch', '--label', 'git co', '-1'])
 */
export async function rawPick(args: string[]): Promise<string | null> {
  const inv = await invokeEngine(['pick', ...args]);
  if (inv.exitCode === 130) {
    return null;
  }
  assertSuccess(inv, 'picktui pick');
  return inv.stdout.replace(/\r?\n$/, '');
}