/**
 * filter / resolve — 纯 TS 过滤匹配引擎（对齐 Go filter.go，零依赖）。
 *
 * 三种匹配模式（与协议一致）：
 *   - 子串 AND（默认）：空格分关键字，全部命中才保留，大小写不敏感
 *   - token 前缀（--sep <chars>）：按分隔符集合切 token（默认 _，可多字符如 ":_"），
 *     每关键字匹配某 token 前缀
 *   - 子序列模糊（--fuzzy）：关键字为子序列
 *
 * 高亮区间为 rune（code point）索引 [start, end)，排序合并；空查询恒 []。
 */
import { toStructuredCands } from './cand.js';
import type { Candidate, FilteredCandidate, FilterMode, FilterOptions } from './types.js';

/** 默认分隔符集合（token 模式）。 */
const DEFAULT_SEP = '_';

/** 将查询字符串拆分为小写关键字列表（空格分词、小写化、过滤空串）。 */
export function splitKeywords(query: string): string[] {
  const fields = query.trim().split(/\s+/).filter(Boolean);
  return fields.map((f) => f.toLowerCase());
}

/** 按 rune 切分字符串为 code point 数组（与 Go []rune 对齐）。 */
function toRunes(s: string): string[] {
  return [...s];
}

/** 某 rune 是否在分隔符集合中。 */
function sepSetOf(sep: string): Set<string> {
  return new Set(toRunes(sep));
}

/** 按分隔符集合切分 runes，返回每个非空 token 的 [start, end) rune 索引区间。 */
function splitRuneRanges(runes: string[], seps: Set<string>): [number, number][] {
  if (seps.size === 0) {
    return runes.length > 0 ? [[0, runes.length]] : [];
  }
  const ranges: [number, number][] = [];
  let start = 0;
  for (let i = 0; i < runes.length; i++) {
    if (seps.has(runes[i])) {
      if (i > start) {
        ranges.push([start, i]);
      }
      start = i + 1;
    }
  }
  if (start < runes.length) {
    ranges.push([start, runes.length]);
  }
  return ranges;
}

/** 按分隔符集合切分字符串为 token 列表（供匹配用）。 */
function splitTokens(s: string, seps: Set<string>): string[] {
  if (seps.size === 0) {
    return [s];
  }
  const tokens: string[] = [];
  let cur = '';
  for (const r of s) {
    if (seps.has(r)) {
      if (cur !== '') {
        tokens.push(cur);
        cur = '';
      }
    } else {
      cur += r;
    }
  }
  if (cur !== '') {
    tokens.push(cur);
  }
  return tokens;
}

/** 子串 AND：所有关键字都需作为子串出现（大小写不敏感）。 */
export function matchSubstringAll(s: string, kws: string[]): boolean {
  const lower = s.toLowerCase();
  return kws.every((kw) => lower.includes(kw));
}

/** token 前缀：按分隔符集合切 token，每关键字须匹配某 token 的前缀。 */
export function matchTokenPrefixAll(s: string, kws: string[], sep: string): boolean {
  const seps = sepSetOf(sep === '' ? DEFAULT_SEP : sep);
  const tokens = splitTokens(s.toLowerCase(), seps);
  return kws.every((kw) => tokens.some((tok) => tok.startsWith(kw)));
}

/** 子序列模糊：所有关键字都需作为子序列出现（大小写不敏感）。 */
export function matchFuzzyAll(s: string, kws: string[]): boolean {
  const lower = s.toLowerCase();
  return kws.every((kw) => isSubsequence(kw, lower));
}

/** pat 是否为 s 的子序列（不要求连续）。 */
export function isSubsequence(pat: string, s: string): boolean {
  if (pat === '') {
    return true;
  }
  let pi = 0;
  for (let si = 0; si < s.length && pi < pat.length; si++) {
    if (s[si] === pat[pi]) {
      pi++;
    }
  }
  return pi === pat.length;
}

/** rune 索引：在 s 中查找 substr 的首个出现位置，未找到返回 -1。 */
function runeIndex(s: string[], substr: string[]): number {
  if (substr.length === 0 || substr.length > s.length) {
    return -1;
  }
  for (let i = 0; i <= s.length - substr.length; i++) {
    let match = true;
    for (let j = 0; j < substr.length; j++) {
      if (s[i + j] !== substr[j]) {
        match = false;
        break;
      }
    }
    if (match) {
      return i;
    }
  }
  return -1;
}

/** 排序并合并重叠的 [start, end) 区间（与 Go mergeRanges 一致）。 */
export function mergeRanges(ranges: [number, number][]): [number, number][] {
  if (ranges.length === 0) {
    return [];
  }
  const sorted = [...ranges].sort((a, b) => (a[0] !== b[0] ? a[0] - b[0] : a[1] - b[1]));
  const merged: [number, number][] = [sorted[0]];
  for (let i = 1; i < sorted.length; i++) {
    const last = merged[merged.length - 1];
    const r = sorted[i];
    if (r[0] <= last[1]) {
      if (r[1] > last[1]) {
        last[1] = r[1];
      }
    } else {
      merged.push(r);
    }
  }
  return merged;
}

/**
 * highlightRanges 返回 s 中匹配关键字的 [start, end) rune 索引区间（已排序合并）。
 * 空查询返回 []。
 */
export function highlightRanges(
  s: string,
  query: string,
  opts: FilterOptions = {},
): [number, number][] {
  const kws = splitKeywords(query);
  if (kws.length === 0) {
    return [];
  }
  const lowerRunes = toRunes(s.toLowerCase());
  const ranges: [number, number][] = [];
  const mode = opts.mode ?? 'substring';

  if (mode === 'token') {
    const seps = sepSetOf(opts.sep === undefined || opts.sep === '' ? DEFAULT_SEP : opts.sep);
    const tokRanges = splitRuneRanges(lowerRunes, seps);
    for (const kw of kws) {
      const kwRunes = toRunes(kw);
      if (kwRunes.length === 0) {
        continue;
      }
      for (const tr of tokRanges) {
        const tok = lowerRunes.slice(tr[0], tr[1]);
        if (hasRunePrefix(tok, kwRunes)) {
          ranges.push([tr[0], tr[0] + kwRunes.length]);
          break;
        }
      }
    }
  } else if (mode === 'fuzzy') {
    for (const kw of kws) {
      const positions = subsequenceRunePositions(toRunes(kw), lowerRunes);
      for (const p of positions) {
        ranges.push([p, p + 1]);
      }
    }
  } else {
    for (const kw of kws) {
      const kwRunes = toRunes(kw);
      if (kwRunes.length === 0) {
        continue;
      }
      const idx = runeIndex(lowerRunes, kwRunes);
      if (idx >= 0) {
        ranges.push([idx, idx + kwRunes.length]);
      }
    }
  }

  return mergeRanges(ranges);
}

function hasRunePrefix(s: string[], prefix: string[]): boolean {
  if (prefix.length > s.length) {
    return false;
  }
  for (let i = 0; i < prefix.length; i++) {
    if (s[i] !== prefix[i]) {
      return false;
    }
  }
  return true;
}

/** 返回 pat 作为 s 子序列匹配时的各字符 rune 位置；不完整匹配返回 []。 */
function subsequenceRunePositions(pat: string[], s: string[]): number[] {
  if (pat.length === 0) {
    return [];
  }
  const positions: number[] = [];
  let pi = 0;
  for (let si = 0; si < s.length && pi < pat.length; si++) {
    if (s[si] === pat[pi]) {
      positions.push(si);
      pi++;
    }
  }
  if (pi < pat.length) {
    return [];
  }
  return positions;
}

/** 匹配模式分派：单候选是否命中全部关键字。 */
export function matchCandidate(s: string, kws: string[], opts: FilterOptions = {}): boolean {
  switch (opts.mode ?? 'substring') {
    case 'token':
      return matchTokenPrefixAll(s, kws, opts.sep ?? '');
    case 'fuzzy':
      return matchFuzzyAll(s, kws);
    default:
      return matchSubstringAll(s, kws);
  }
}

export interface FilterOptsNormalized {
  mode: FilterMode;
  sep: string;
}

/** 归一化 FilterOptions（对齐协议 --mode/--sep 语义）。 */
export function normalizeOpts(opts: FilterOptions = {}): FilterOptsNormalized {
  return { mode: opts.mode ?? 'substring', sep: opts.sep ?? '' };
}

/**
 * filterCands — 过滤 + 高亮区间（纯同步，保留输入顺序）。
 * 描述不参与过滤——过滤永远针对"选中什么"而非"展示什么"。空查询返回全部。
 */
export function filterCands(
  cands: Candidate[],
  query: string,
  opts: FilterOptions = {},
): FilteredCandidate[] {
  const kws = splitKeywords(query);
  if (kws.length === 0) {
    return cands.map((c) => ({ value: c.value, desc: c.desc ?? '', ranges: [] }));
  }
  const norm = normalizeOpts(opts);
  const out: FilteredCandidate[] = [];
  for (const c of cands) {
    if (matchCandidate(c.value, kws, norm)) {
      out.push({
        value: c.value,
        desc: c.desc ?? '',
        ranges: highlightRanges(c.value, query, norm),
      });
    }
  }
  return out;
}

/**
 * autoResolve — 把查询字符串解析为唯一候选（供 --auto 自动选中）：
 *   1. 精确匹配（大小写敏感）直接命中；
 *   2. 否则大小写不敏感的唯一前缀匹配；
 * 无匹配或多个前缀匹配时返回 null。
 */
export function autoResolve(cands: Candidate[], query: string): string | null {
  if (query === '') {
    return null;
  }
  for (const c of cands) {
    if (c.value === query) {
      return c.value;
    }
  }
  const lq = query.toLowerCase();
  let only: string | null = null;
  for (const c of cands) {
    if (c.value.toLowerCase().startsWith(lq)) {
      if (only !== null && only !== c.value) {
        return null; // 多个前缀匹配，无法唯一确定
      }
      only = c.value;
    }
  }
  return only;
}

/**
 * filter — 过滤候选并返回命中项（保留输入顺序）+ 高亮区间。
 * 与协议 filter --json 等价；空查询返回全部（ranges 恒为 []）。
 */
export function filter(
  cands: (string | Candidate)[],
  query = '',
  opts: FilterOptions = {},
): Promise<FilteredCandidate[]> {
  return Promise.resolve(filterCands(toStructuredCands(cands), query, opts));
}

/**
 * resolve — 把 query 解析为唯一候选（精确 → 唯一前缀，大小写不敏感前缀）。
 * @returns 解析出的选中值；无匹配/多个前缀匹配/空 query 返回 null
 */
export function resolve(
  cands: (string | Candidate)[],
  query: string,
  _opts: FilterOptions = {},
): Promise<string | null> {
  return Promise.resolve(autoResolve(toStructuredCands(cands), query));
}