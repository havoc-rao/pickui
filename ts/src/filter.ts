/**
 * filter / resolve — 纯函数数据面（直调引擎 filter/resolve --json）。
 *
 * 过滤与高亮区间由引擎权威计算；本模块只做「候选传入、JSON 取回」，
 * 禁止重实现匹配（详见 docs/protocol.md 兼容性承诺）。
 */
import {
  assertSuccess,
  invokeEngine,
  parseJSON,
  serializeCandidates,
} from './engine.js';
import type { Candidate, FilterOptions, FilteredCandidate, ResolveResult } from './types.js';

/** 将 FilterOptions 转为引擎参数。 */
function modeArgs(opts: FilterOptions): string[] {
  const args: string[] = [];
  if (opts.mode && opts.mode !== 'substring') {
    args.push('--mode', opts.mode);
  }
  if (opts.mode === 'token' && opts.sep) {
    args.push('--sep', opts.sep);
  }
  return args;
}

/**
 * filter — 过滤候选并返回命中项（保留输入顺序）+ 高亮区间。
 *
 * @param cands 候选（字符串或结构化 Candidate）
 * @param query 过滤关键字（空格分词，空串返回全部）
 * @param opts  匹配模式（默认 substring）
 * @returns 命中候选，`ranges` 为命中的 rune 区间（空查询恒为 []）
 */
export async function filter(
  cands: (string | Candidate)[],
  query = '',
  opts: FilterOptions = {},
): Promise<FilteredCandidate[]> {
  const inv = await invokeEngine(
    ['filter', '--json', '--query', query, ...modeArgs(opts)],
    { input: serializeCandidates(cands) },
  );
  assertSuccess(inv, 'pickui filter');
  return parseJSON<FilteredCandidate[]>(inv.stdout, 'pickui filter');
}

/**
 * resolve — 把 query 解析为唯一候选（精确 → 唯一前缀，大小写不敏感前缀）。
 *
 * @returns 解析出的选中值；无匹配/多个前缀匹配返回 null（引擎退出码 1）
 */
export async function resolve(
  cands: (string | Candidate)[],
  query: string,
  opts: FilterOptions = {},
): Promise<string | null> {
  const inv = await invokeEngine(
    ['resolve', '--json', '--query', query, ...modeArgs(opts)],
    { input: serializeCandidates(cands) },
  );
  if (inv.exitCode === 1) {
    return null; // 协议：无唯一解 → stdout 空 + 退出码 1
  }
  assertSuccess(inv, 'pickui resolve');
  const parsed = parseJSON<ResolveResult>(inv.stdout, 'pickui resolve');
  return parsed.value as string;
}