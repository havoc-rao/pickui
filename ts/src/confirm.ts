/**
 * confirm — 自动匹配的首次用户确认记录（confirm.toml）。
 *
 * 文件格式与 Go 引擎 BurntSushi/toml 输出完全兼容：
 *
 *	# picktui confirm — user-confirmed auto resolutions (auto-managed)
 *	[confirmed]
 *	"npm run" = ["release", "dev"]
 *
 * label 为空时不做任何落盘（返回 false）。文件不存在或损坏 → 空记录
 * （确认记录只是安全辅助，缺失最多导致首次匹配再次询问）。
 */
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

import { dataDir } from './config.js';
import { parseToml, escapeTomlString, keyLiteral } from './toml.js';
import { PicktuiError } from './types.js';

/** 自动匹配确认记录文件名。 */
export const CONFIRM_FILE = 'confirm.toml';

/** 注释头（对齐引擎输出）。 */
const HEADER = '# picktui confirm — user-confirmed auto resolutions (auto-managed)';

/** 返回确认记录文件路径。 */
export function confirmPath(): string {
  return join(dataDir(), CONFIRM_FILE);
}

/** 解码 confirm.toml 为 label → 已确认值列表；缺失/损坏时返回空 Map。 */
export function loadConfirmed(): Map<string, string[]> {
  const p = confirmPath();
  if (!existsSync(p)) {
    return new Map();
  }
  const doc = parseToml(readFileSync(p, 'utf8'));
  if (doc === null) {
    return new Map();
  }
  const confirmed = doc.get('confirmed');
  if (!confirmed) {
    return new Map();
  }
  const out = new Map<string, string[]>();
  for (const [label, values] of confirmed) {
    if (Array.isArray(values)) {
      out.set(label, values.map(String));
    }
  }
  return out;
}

/** 报告 (label, value) 是否已被确认过；label 为空时恒为 false。 */
export function isConfirmed(label: string, value: string): boolean {
  if (label === '') {
    return false;
  }
  return (loadConfirmed().get(label) ?? []).includes(value);
}

/** 记录 (label, value) 为已确认（幂等；label 为空不落盘）。 */
export function confirm(label: string, value: string): void {
  if (label === '') {
    return;
  }
  const m = loadConfirmed();
  const list = m.get(label) ?? [];
  if (list.includes(value)) {
    return;
  }
  list.push(value);
  m.set(label, list);

  const lines: string[] = [HEADER, '[confirmed]'];
  const labels = [...m.keys()].sort();
  for (const l of labels) {
    const values = (m.get(l) ?? []).map(escapeTomlString).map((v) => `"${v}"`);
    lines.push(`${keyLiteral(l)} = [${values.join(', ')}]`);
  }
  const p = confirmPath();
  try {
    mkdirSync(dirname(p), { recursive: true });
    writeFileSync(p, lines.join('\n') + '\n', { mode: 0o644 });
  } catch (err) {
    throw new PicktuiError(
      `写入确认记录失败：${err instanceof Error ? err.message : String(err)}`,
      1,
      '',
    );
  }
}

/** 解析确认回答：y/yes（大小写不敏感，容忍首尾空白）为确认。 */
export function parseConfirmAnswer(s: string): boolean {
  const t = s.trim().toLowerCase();
  return t === 'y' || t === 'yes';
}

/** 报告 (label, value) 是否已被确认过。 */
export async function confirmCheck(label: string, value: string): Promise<boolean> {
  return isConfirmed(label, value);
}

/** 记录 (label, value) 为已确认（幂等）；返回确认后的实际状态。 */
export async function confirmAdd(label: string, value: string): Promise<boolean> {
  confirm(label, value);
  return isConfirmed(label, value);
}