/**
 * config — 数据目录解析（对齐协议：hist/confirm/pick --label 状态文件所在）。
 *
 * 优先级：$PICKTUI_CONFIG_DIR → $XDG_CONFIG_HOME/picktui → ~/.config/picktui。
 */
import { homedir } from 'node:os';
import { join } from 'node:path';

/** 状态文件目录（history.toml / confirm.toml 所在）。 */
export function dataDir(): string {
  const explicit = process.env.PICKTUI_CONFIG_DIR;
  if (explicit) {
    return explicit;
  }
  const base = process.env.XDG_CONFIG_HOME || join(homedir(), '.config');
  return join(base, 'picktui');
}