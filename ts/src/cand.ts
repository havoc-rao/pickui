/**
 * cand — 结构化候选：key<TAB>description 拆分与清洗（纯函数，对齐协议总览）。
 *
 * 过滤只匹配 key，TUI 展示 description，选中输出 key——"展示与选中分离"。
 * 行为与 Go 引擎 StructuredCandidates 完全一致：
 *   1. 先按 TAB 拆 key<TAB>description（无 TAB 则整行为 key）；
 *   2. key 部分 trim → 去 "* "/"+ " 前缀（git branch 标记）→ 再 trim；
 *   3. 跳空 key；按 key 去重保序——后到行可为先到同名 key 补缺失描述；
 *   4. description 部分 trim（key 为空的行——如行首 TAB——直接跳过）。
 */
import type { Candidate } from './types.js';

/** 将 `string | Candidate` 输入序列化为协议结构化行（key<TAB>description）。 */
export function serializeLines(cands: (string | Candidate)[]): string[] {
  return cands.map((c) => (typeof c === 'string' ? c : c.desc ? `${c.value}\t${c.desc}` : c.value));
}

/** 将原始行列表解析为结构化候选（与 Go StructuredCandidates 同语义）。 */
export function structuredCandidates(lines: string[]): Candidate[] {
  const idx = new Map<string, number>();
  const out: Candidate[] = [];
  for (const line of lines) {
    let value = line;
    let desc = '';
    const tab = line.indexOf('\t');
    if (tab >= 0) {
      value = line.slice(0, tab);
      desc = line.slice(tab + 1);
    }
    value = value.trim();
    if (value === '') {
      continue;
    }
    if (value.startsWith('* ')) {
      value = value.slice(2);
    } else if (value.startsWith('+ ')) {
      value = value.slice(2);
    }
    value = value.trim();
    if (value === '') {
      continue;
    }
    desc = desc.trim();
    const j = idx.get(value);
    if (j !== undefined) {
      if (out[j].desc === '' && desc !== '') {
        out[j] = { value, desc };
      }
      continue;
    }
    idx.set(value, out.length);
    out.push({ value, desc });
  }
  return out;
}

/** 输入候选统一解析为结构化候选（filter/resolve/pick 共用入口）。 */
export function toStructuredCands(cands: (string | Candidate)[]): Candidate[] {
  return structuredCandidates(serializeLines(cands));
}