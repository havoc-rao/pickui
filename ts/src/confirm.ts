/**
 * confirm — 自动匹配首次确认（文件状态，直调引擎 confirm 子命令）。
 *
 * 状态文件位于引擎数据目录，与 pick --auto 的确认记录同一份：
 * 首次自动匹配某 (label, value) 需确认，确认后不再询问。
 */
import { assertSuccess, invokeEngine, parseJSON } from './engine.js';

/** 报告 (label, value) 是否已被确认过。 */
export async function confirmCheck(label: string, value: string): Promise<boolean> {
  const inv = await invokeEngine(['confirm', 'check', label, value]);
  assertSuccess(inv, 'picktui confirm check');
  const parsed = parseJSON<{ confirmed: boolean }>(inv.stdout, 'picktui confirm check');
  return parsed.confirmed;
}

/**
 * 记录 (label, value) 为已确认（幂等）。
 * @returns 确认后的实际状态（label 为空时不落盘 → false）
 */
export async function confirmAdd(label: string, value: string): Promise<boolean> {
  const inv = await invokeEngine(['confirm', 'add', label, value]);
  assertSuccess(inv, 'picktui confirm add');
  const parsed = parseJSON<{ confirmed: boolean }>(inv.stdout, 'picktui confirm add');
  return parsed.confirmed;
}