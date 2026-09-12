/**
 * tui — pick / menu / rawPick（纯 TS 实现，零引擎依赖）。
 *
 * 交互渲染与按键输入走 /dev/tty（posix）或宿主 stdio（Windows fallback），
 * 宿主 stdout 只承载选中值——与引擎协议相同的安全语义。无 TTY 时自动退化
 * （无 query 取首个、有 query 过滤取首），调用方无需分支；取消返回 null。
 *
 * 流程对齐 Go 引擎 pickWith/menuWith：--auto 确认链、-1 自动选中、
 * 非交互开关（$PICKTUI_PICK=off）、选择记忆（--label）、候选来源
 * （位置参数/stdin/--from/--map）。
 */
import { exec } from 'node:child_process';
import { existsSync } from 'node:fs';
import { homedir } from 'node:os';
import { extname, join } from 'node:path';

import { toStructuredCands } from './cand.js';
import { confirm, isConfirmed } from './confirm.js';
import { autoResolve, filterCands } from './filter.js';
import { lastPick, loadPickHistory, savePick } from './history.js';
import {
  newModel,
  selectedValues,
  setColorEnabled,
  update,
  view,
  type KeyLike,
  type ModelOpts,
  type UiModel,
} from './model.js';
import {
  KeyReader,
  enterInteractive,
  exitInteractive,
  openTty,
  promptConfirm,
  renderFrame,
  type Key,
  type Tty,
} from './tty.js';
import type { Candidate, FilterOptions, PickFlags } from './types.js';
import { PicktuiError } from './types.js';

/** 命令名前缀（对齐引擎默认名，宿主品牌对齐不在本包范围）。 */
const NAME = 'picktui';

/** 非交互开关环境变量（值为 "off" 时 pick 跳过 TUI）。 */
export const OFF_ENV = 'PICKTUI_PICK';

/** shell 候选来源/转换器超时（对齐引擎 10s）。 */
const CMD_TIMEOUT_MS = 10_000;

/** 光标闪烁周期（对齐引擎 500ms）。 */
const TICK_MS = 500;

/**
 * pick — 交互过滤选择（fzf 风格）。
 *
 * @param cands 候选（不传时从宿主 stdin 管道读取）
 * @param flags query/label/sep/fuzzy/select1/auto
 * @returns 选中值；用户取消（esc/ctrl+c）返回 null
 */
export async function pick(
  cands: (string | Candidate)[] = [],
  flags: PickFlags = {},
): Promise<string | null> {
  const structured = cands.length > 0 ? toStructuredCands(cands) : [];
  return pickFlow({
    cands: structured,
    fromCmd: '',
    mapper: '',
    flags,
  });
}

/**
 * menu — 多值缩写 TUI 菜单（数字 1-9 直选）。
 * @returns 选中值；取消返回 null；参数不足抛 PicktuiError（退出码 2）
 */
export async function menu(label: string, cands: string[]): Promise<string | null> {
  if (cands.length < 1) {
    const msg = `usage: ${NAME} _menu <label> <cand...>`;
    throw new PicktuiError(msg, 2, msg);
  }
  const tty = openTty();
  if (tty === null) {
    // 非交互：取首个候选（对齐引擎退化：保证脚本不卡）
    return cands[0];
  }
  try {
    const res = await runTui(
      newModel({
        title: `${NAME} ${label}`,
        subtitle: '多值缩写',
        cands: cands.map((v) => ({ value: v, desc: '' })),
        query: '',
        filterOpts: { mode: 'substring' },
        initial: memoryInitial(label, cands.map((v) => ({ value: v, desc: '' }))),
        jumpKeys: true,
        statusHint: 'type to filter · ↑↓ move · 1-9 jump · enter confirm · esc cancel',
        multi: false,
        selected: [],
      }),
      tty,
    );
    if (res.cancelled) {
      return null;
    }
    if (label !== '') {
      savePick(label, res.value);
    }
    return res.value;
  } finally {
    tty.read.destroy();
  }
}

/**
 * rawPick — 透传引擎 pick 参数（--from/--map/-q/--label/--sep/--fuzzy/-1/--auto）。
 * @example rawPick(['--from', 'git branch', '--label', 'git co', '-1'])
 */
export async function rawPick(args: string[]): Promise<string | null> {
  const parsed = parsePickArgs(args);
  const structured = parsed.rest.length > 0 ? toStructuredCands(parsed.rest) : [];
  return pickFlow({
    cands: structured,
    fromCmd: parsed.from,
    mapper: parsed.mapper,
    flags: parsed.flags,
  });
}

interface ParsedPickArgs {
  flags: PickFlags;
  from: string;
  mapper: string;
  rest: string[];
}

/** 解析 pick 参数（对齐引擎 flags：--query=、--from=、--map= 等长格式均支持）。 */
function parsePickArgs(args: string[]): ParsedPickArgs {
  const flags: PickFlags = {};
  let from = '';
  let mapper = '';
  const rest: string[] = [];
  for (let i = 0; i < args.length; i++) {
    const a = args[i];
    const take = (name: string): string => {
      if (i + 1 < args.length) {
        i++;
        return args[i];
      }
      throw new PicktuiError(`${NAME} pick: ${name} 缺少参数`, 2, '');
    };
    if (a === '-q' || a === '--query') {
      flags.query = take('--query');
    } else if (a.startsWith('--query=')) {
      flags.query = a.slice('--query='.length);
    } else if (a === '--from') {
      from = take('--from');
    } else if (a.startsWith('--from=')) {
      from = a.slice('--from='.length);
    } else if (a === '--map') {
      mapper = take('--map');
    } else if (a.startsWith('--map=')) {
      mapper = a.slice('--map='.length);
    } else if (a === '--label') {
      flags.label = take('--label');
    } else if (a.startsWith('--label=')) {
      flags.label = a.slice('--label='.length);
    } else if (a === '--sep') {
      flags.sep = take('--sep');
    } else if (a.startsWith('--sep=')) {
      flags.sep = a.slice('--sep='.length);
    } else if (a === '--fuzzy') {
      flags.fuzzy = true;
    } else if (a === '-1' || a === '--select-1') {
      flags.select1 = true;
    } else if (a === '--auto') {
      flags.auto = true;
    } else if (a.startsWith('-') && a !== '-') {
      const msg = `${NAME} pick: 未知参数 ${a}`;
      throw new PicktuiError(msg, 2, msg);
    } else {
      rest.push(a);
    }
  }
  return { flags, from, mapper, rest };
}

interface PickFlowParams {
  cands: Candidate[];
  fromCmd: string;
  mapper: string;
  flags: PickFlags;
}

/** pick 全流程（对齐引擎 pickWith）。 */
async function pickFlow(params: PickFlowParams): Promise<string | null> {
  const { flags, fromCmd, mapper } = params;
  const query = flags.query ?? '';

  if (mapper !== '' && params.cands.length > 0) {
    const msg = `${NAME} pick: --map 只与 --from 配合，不能与位置参数同用`;
    throw new PicktuiError(msg, 2, msg);
  }

  // 构造 FilterOptions（对齐引擎：--fuzzy 优先，--sep 切 token 模式）
  const filterOpts: FilterOptions = { mode: 'substring' };
  if (flags.fuzzy) {
    filterOpts.mode = 'fuzzy';
  } else if (flags.sep) {
    filterOpts.mode = 'token';
    filterOpts.sep = flags.sep;
  }

  // 候选来源：显式候选（参数）或 --from 命令执行。
  // 注意：不读宿主进程自身的 stdin——宿主 stdin 是宿主自己的输入流，
  // 由库消费会造成半开 pipe 阻塞并侵入宿主输入（对齐旧版绑定行为）。
  let cands = params.cands;
  if (fromCmd !== '') {
    const raw = await runShell(fromCmd, '');
    if (!raw.ok) {
      throw new PicktuiError(`${NAME} pick: ${fromCmd}: ${raw.stderr}`, 1, raw.stderr);
    }
    cands = toStructuredCands(raw.stdout.split('\n'));
    // --map 转换器：接收 --from 原始输出
    if (mapper !== '') {
      cands = await runMapper(mapper, raw.stdout);
    }
  }
  if (cands.length === 0) {
    const msg = `${NAME} pick: no candidates`;
    throw new PicktuiError(msg, 1, msg);
  }

  const label = flags.label ?? '';
  const off = process.env[OFF_ENV] === 'off';

  // --auto 自动选中：query 精确命中或唯一前缀命中时直接输出（跳过 TUI）。
  // 首次匹配该 (label, value) 需用户确认（记录 confirm 记录，之后不再询问）；
  // off / 无 tty / 无 label 时跳过确认；拒绝确认回退下方 TUI。
  if (flags.auto && query !== '') {
    const v = autoResolve(cands, query);
    if (v !== null) {
      if (label === '' || off || isConfirmed(label, v)) {
        savePickIfLabeled(label, v);
        return v;
      }
      const tty = openTty();
      if (tty !== null) {
        try {
          const answer = await promptConfirm(
            tty,
            `\r\n${NAME}: 首次自动匹配 ${label} → ${v}，确认执行？[y/N] `,
          );
          if (answer !== null && /^(y|yes)$/i.test(answer.trim())) {
            confirm(label, v);
            savePickIfLabeled(label, v);
            return v;
          }
          // 用户拒绝确认：回退到下方 TUI 选择（query 预填过滤）
        } finally {
          tty.read.destroy();
        }
      } else {
        // 无 /dev/tty：非交互环境跳过确认，保持确定性执行
        savePickIfLabeled(label, v);
        return v;
      }
    }
  }

  // -1 自动选中（仅一个匹配时）
  if (flags.select1) {
    const filtered = filterCands(cands, query, filterOpts);
    if (filtered.length === 1) {
      savePickIfLabeled(label, filtered[0].value);
      return filtered[0].value;
    }
  }

  // 非交互开关或无 tty → 退化路径
  const tty = openTty();
  if (off || tty === null) {
    return pickNonInteractive(cands, query, filterOpts, label, flags.auto === true);
  }

  try {
    const res = await runTui(
      newModel({
        title: `${NAME} pick`,
        subtitle: label,
        cands,
        query,
        filterOpts,
        initial: memoryInitial(label, cands),
        jumpKeys: false,
        statusHint: 'type to filter · ↑↓ move · enter select · esc cancel',
        multi: false,
        selected: [],
      }),
      tty,
    );
    if (res.cancelled) {
      return null;
    }
    // 自动解析流程中用户从 TUI 显式选中 → 视为对该解析的确认
    if (flags.auto && label !== '') {
      confirm(label, res.value);
    }
    savePickIfLabeled(label, res.value);
    return res.value;
  } finally {
    tty.read.destroy();
  }
}

/** 非交互模式（对齐引擎 pickNonInteractive）。 */
function pickNonInteractive(
  cands: Candidate[],
  query: string,
  opts: FilterOptions,
  label: string,
  auto: boolean,
): string | null {
  if (auto && query !== '') {
    const v = autoResolve(cands, query);
    if (v !== null) {
      savePickIfLabeled(label, v);
      return v;
    }
    const msg = `${NAME} pick: no unique match for "${query}"`;
    throw new PicktuiError(msg, 1, msg);
  }
  if (query === '') {
    const chosen = cands[0].value;
    savePickIfLabeled(label, chosen);
    return chosen;
  }
  const filtered = filterCands(cands, query, opts);
  if (filtered.length === 0) {
    const msg = `${NAME} pick: no match for "${query}"`;
    throw new PicktuiError(msg, 1, msg);
  }
  const chosen = filtered[0].value;
  savePickIfLabeled(label, chosen);
  return chosen;
}

/** label 非空时记录选择记忆（失败不中断主流程，与引擎一致）。 */
function savePickIfLabeled(label: string, value: string): void {
  if (label === '') {
    return;
  }
  try {
    savePick(label, value);
  } catch {
    // 记忆只是锦上添花，记录失败不影响选中结果
  }
}

/** 选择记忆：仅 query 为空时应用（有预填过滤词说明是精确搜索，不记忆）。 */
function memoryInitial(label: string, cands: Candidate[]): number {
  if (label === '') {
    return 0;
  }
  try {
    const last = lastPick(loadPickHistory(), label);
    if (last !== '') {
      for (let i = 0; i < cands.length; i++) {
        if (cands[i].value === last) {
          return i;
        }
      }
    }
  } catch {
    // 记忆损坏时静默回退首行
  }
  return 0;
}

/** 在 tty 上运行 TUI 选择器，返回（取消, 选中值, 多选勾选）。 */
export async function runTui(
  model: UiModel,
  tty: Tty,
): Promise<{ cancelled: boolean; value: string; selected: string[] }> {
  enterInteractive(tty);
  const reader = new KeyReader(tty.read);
  const onResize = () => {
    model.width = tty.write.columns > 0 ? tty.write.columns : model.width;
    model.height = tty.write.rows > 0 ? tty.write.rows : model.height;
  };
  tty.write.on('resize', onResize);
  onResize();

  let pending: Promise<Key | null> | null = null;
  const nextKey = () => {
    if (pending === null) {
      pending = reader.next().finally(() => {
        pending = null;
      });
    }
    return pending;
  };

  try {
    for (;;) {
      const [kind, value] = await Promise.race([
        nextKey().then((k) => ['key', k] as const),
        sleep(TICK_MS).then(() => ['tick', null] as const),
      ]);
      if (kind === 'key') {
        const k = value as Key | null;
        if (k === null) {
          break; // 输入流关闭（EOF）
        }
        model = update(model, toKeyLike(k));
        if (model.quit) {
          break;
        }
      } else {
        model = { ...model, phase: model.phase ^ 1 };
      }
      renderFrame(tty, view(model));
    }
  } finally {
    exitInteractive(tty);
    tty.write.removeListener('resize', onResize);
    tty.write.write('\x1b[2J\x1b[H');
  }

  if (model.cancelled || model.filtered.length === 0) {
    return { cancelled: true, value: '', selected: [] };
  }
  if (model.opts.multi) {
    return {
      cancelled: false,
      value: model.filtered[model.cursor].value,
      selected: selectedValues(model),
    };
  }
  return { cancelled: false, value: model.filtered[model.cursor].value, selected: [] };
}

/** 归一化按键：可打印字符转字符串（'k'/'j' 等移动键语义由 model 处理）。 */
function toKeyLike(k: Key): KeyLike {
  if (typeof k === 'object') {
    return k.value === '' ? 'x' : k.value; // 左右方向键等忽略键映射为无操作
  }
  return k;
}

/** 执行 shell 命令（10s 超时，input 喂 stdin）；失败时附带 stderr 诊断。 */
function runShell(
  cmd: string,
  input: string,
): Promise<{ ok: true; stdout: string } | { ok: false; stderr: string }> {
  return new Promise((resolve) => {
    const child = exec(
      cmd,
      {
        timeout: CMD_TIMEOUT_MS,
        maxBuffer: 64 * 1024 * 1024,
        windowsHide: true,
      },
      (err, stdout) => {
        if (err) {
          const stderr = (err as Error & { stderr?: string }).stderr ?? '';
          resolve({ ok: false, stderr: stderr.trim() || err.message });
          return;
        }
        resolve({ ok: true, stdout: stdout ?? '' });
      },
    );
    child.stdin?.end(input);
  });
}

/** 执行 --map 转换器：stdin 喂 input，输出按结构化候选解析；脚本缺失 fail-fast。 */
async function runMapper(mapper: string, input: string): Promise<Candidate[]> {
  const missing = mapperScriptMissing(mapper);
  if (missing !== '') {
    const msg = `map 脚本文件不存在: ${missing}\n  → 脚本可能已被移动、删除或重命名`;
    throw new PicktuiError(msg, 1, msg);
  }
  const res = await runShell(mapper, input);
  if (!res.ok) {
    throw new PicktuiError(`${NAME} pick: --map ${mapper}: ${res.stderr}`, 1, res.stderr);
  }
  return toStructuredCands(res.stdout.split('\n'));
}

/**
 * 扫描 mapper 命令，识别"解释器 + 脚本路径"形态，返回第一个不存在的脚本文件路径。
 * 保守策略：跳过 flags/引号/变量/管道等 shell 语法，只检查形如路径的 token
 * （/ ./ ../ ~/ 前缀、带常见脚本扩展名），避免误拦 node -e / python3 -m 等合法用法。
 */
export function mapperScriptMissing(mapper: string): string {
  const fields = mapper.trim().split(/\s+/).filter(Boolean);
  if (fields.length < 2) {
    return '';
  }
  for (const tok of fields.slice(1)) {
    if (tok.startsWith('-')) {
      continue;
    }
    if (/['"`${}|;&<>?*=]/.test(tok)) {
      continue;
    }
    if (!scriptPathLike(tok)) {
      continue;
    }
    const p = expandTilde(tok);
    if (existsSync(p)) {
      continue;
    }
    return p;
  }
  return '';
}

/** token 是否形如脚本文件路径：/ ./ ../ ~/ 前缀，或带常见脚本扩展名。 */
function scriptPathLike(tok: string): boolean {
  if (tok.startsWith('/') || tok.startsWith('./') || tok.startsWith('../') || tok.startsWith('~/')) {
    return true;
  }
  return isScriptExt(extname(tok).toLowerCase());
}

/** 常见脚本解释器可直接执行的文件扩展名。 */
function isScriptExt(ext: string): boolean {
  return [
    '.mjs', '.cjs', '.js', '.ts', '.py', '.sh', '.bash', '.zsh', '.fish',
    '.pl', '.rb', '.php', '.lua', '.exs', '.r',
  ].includes(ext);
}

/** 展开 ~ 与 ~/ 前缀为用户主目录。 */
function expandTilde(p: string): string {
  if (p === '~') {
    return homedir();
  }
  if (p.startsWith('~/')) {
    return join(homedir(), p.slice(2));
  }
  return p;
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

// 颜色开关：NO_COLOR 环境变量时降级纯文本
if (process.env.NO_COLOR !== undefined && process.env.NO_COLOR !== '') {
  setColorEnabled(false);
}