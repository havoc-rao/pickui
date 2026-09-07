/**
 * history — 选择记忆（文件状态，直调引擎 hist 子命令）。
 *
 * 状态文件位于引擎数据目录（$PICKUI_CONFIG_DIR → $XDG_CONFIG_HOME/pickui →
 * ~/.config/pickui），与 pick --label 的 TUI 记忆同一份。
 */
import { assertSuccess, invokeEngine, parseJSON } from './engine.js';

/** 读取 label 上次选中的值；无记录返回 ""。 */
export async function histGet(label: string): Promise<string> {
  const inv = await invokeEngine(['hist', 'get', label]);
  assertSuccess(inv, 'pickui hist get');
  const parsed = parseJSON<{ label: string; last: string }>(inv.stdout, 'pickui hist get');
  return parsed.last ?? '';
}

/** 记录 label 的上次选中值（成功无输出；失败抛 PickuiError）。 */
export async function histSet(label: string, value: string): Promise<void> {
  const inv = await invokeEngine(['hist', 'set', label, value]);
  assertSuccess(inv, 'pickui hist set');
}