/**
 * history — 选择记忆：记住每个选择器/歧义点上次选中的候选（history.toml）。
 *
 * 文件格式与 Go 引擎 BurntSushi/toml 输出完全兼容（同一数据目录共用）：
 *
 *	# picktui history — last selection per key (auto-managed)
 *	["git p"]
 *	last = "pull"
 *
 * 曾用名 picks.toml 仍会被自动迁移到新位置（与引擎行为一致）。
 * 文件不存在或损坏 → 空记录（记忆只是锦上添花，静默容错）。
 */
import { existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

import { dataDir } from './config.js';
import { parseToml, escapeTomlString, keyLiteral } from './toml.js';
import { PicktuiError } from './types.js';

/** 选择记忆文件名；曾用名 picks.toml 因易与 pick 命令混淆而弃用。 */
export const HISTORY_FILE = 'history.toml';

/** 旧版文件名，仅用于首次读取时自动迁移。 */
export const LEGACY_HISTORY_FILE = 'picks.toml';

/** 注释头（对齐引擎输出，便于用户辨识文件来源）。 */
const HEADER = '# picktui history — last selection per key (auto-managed)';

/** 返回选择记忆文件路径。 */
export function historyPath(): string {
  return join(dataDir(), HISTORY_FILE);
}

/** history.toml 原始内容（label → { last }），与引擎同一份。 */
export function loadPickHistory(): Map<string, string> {
  const doc = loadHistoryToml();
  const out = new Map<string, string>();
  for (const label of doc.keys()) {
    const table = doc.get(label);
    if (!table) {
      continue;
    }
    const last = table.get('last');
    if (typeof last === 'string') {
      out.set(label, last);
    }
  }
  return out;
}

/** 读取/迁移 history.toml：不存在时静默返回空 Map。 */
function loadHistoryToml(): Map<string, Map<string, string | string[]>> {
  const p = historyPath();
  if (existsSync(p)) {
    return parseHistoryFile(readFileSync(p, 'utf8'));
  }
  const legacy = join(dataDir(), LEGACY_HISTORY_FILE);
  if (existsSync(legacy)) {
    const raw = parseHistoryFile(readFileSync(legacy, 'utf8'));
    try {
      writeHistoryEntries(raw);
      rmSync(legacy, { force: true });
    } catch {
      // 迁移写入失败不阻断读取——旧数据仍在内存中可用
    }
    return raw;
  }
  return new Map();
}

/** 解析 history.toml 文本；损坏时静默返回空（对齐引擎 LoadPickHistory）。 */
function parseHistoryFile(text: string): Map<string, Map<string, string | string[]>> {
  const doc = parseToml(text);
  if (doc === null) {
    return new Map();
  }
  return doc;
}

/** 取回 label 上次选中的值；无记录返回 ""。 */
export function lastPick(raw: Map<string, string>, label: string): string {
  return raw.get(label) ?? '';
}

/** 记录 label 的上次选中值（保留其它 label 的记忆后写回）。 */
export function savePick(label: string, value: string): void {
  const raw = loadPickHistory();
  raw.set(label, value);
  writeHistoryEntries(raw);
}

/** 将记忆写回 history.toml（自动建目录，对齐引擎输出格式）。 */
function writeHistoryEntries(
  entries: Map<string, Map<string, string | string[]>> | Map<string, string>,
): void {
  const lines: string[] = [HEADER];
  const labels = [...entries.keys()].sort();
  for (const label of labels) {
    lines.push(`[${keyLiteral(label)}]`);
    const table = entries.get(label);
    const last = table instanceof Map ? (table.get('last') as string | undefined) : table;
    lines.push(`last = "${escapeTomlString(last ?? '')}"`);
  }
  const p = historyPath();
  try {
    mkdirSync(dirname(p), { recursive: true });
    writeFileSync(p, lines.join('\n') + '\n', { mode: 0o644 });
  } catch (err) {
    throw new PicktuiError(
      `写入选择记忆失败：${err instanceof Error ? err.message : String(err)}`,
      1,
      '',
    );
  }
}

/** 读取 label 上次选中的值；无记录返回 ""。 */
export async function histGet(label: string): Promise<string> {
  return lastPick(loadPickHistory(), label);
}

/** 记录 label 的上次选中值（失败抛 PicktuiError，含退出码 1）。 */
export async function histSet(label: string, value: string): Promise<void> {
  savePick(label, value);
}