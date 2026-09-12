/**
 * tty — 终端抽象（纯 Node 标准库，无第三方依赖）。
 *
 * 协议约定：交互 TUI 独立于宿主 stdout——posix 上打开 /dev/tty 渲染与输入，
 * 宿主 stdout 只承载选中值（$(...) 安全）。无 /dev/tty 时（如 Windows）退化为
 * 宿主自身的 stdio（isTTY 时才可用）；两者皆不可用时返回 null，由调用方走
 * 非交互退化路径（与引擎行为一致）。
 *
 * 按键输入：raw mode + ANSI 转义序列解析（方向键/回车/tab/ctrl 组合/UTF-8 字符）。
 */
import { openSync } from 'node:fs';
import { StringDecoder } from 'node:string_decoder';
import { ReadStream, WriteStream } from 'node:tty';

/** 交互终端句柄。 */
export interface Tty {
  read: ReadStream;
  write: WriteStream;
  /** 终端尺寸（列）。 */
  columns: number;
  /** 终端尺寸（行）。 */
  rows: number;
}

/** 按键（对齐引擎 model 的键位集合；数字作为字符处理）。 */
export type Key =
  | 'enter'
  | 'esc'
  | 'up'
  | 'down'
  | 'tab'
  | 'backspace'
  | 'ctrl+u'
  | 'ctrl+w'
  | 'ctrl+c'
  | 'ctrl+p'
  | 'ctrl+n'
  | 'space'
  | { type: 'char'; value: string };

/** 尝试打开交互终端；不可用返回 null（调用方退化为非交互）。 */
export function openTty(): Tty | null {
  // POSIX：/dev/tty 独立于宿主 stdout/stderr
  try {
    const fd = openSync('/dev/tty', 'r+');
    const read = new ReadStream(fd);
    const write = new WriteStream(fd);
    return {
      read,
      write,
      columns: write.columns > 0 ? write.columns : 80,
      rows: write.rows > 0 ? write.rows : 24,
    };
  } catch {
    // Windows 等无 /dev/tty：使用宿主 stdio（仅当确为交互终端）
    if (process.stdin.isTTY && process.stdout.isTTY) {
      return {
        read: process.stdin,
        write: process.stdout,
        columns: process.stdout.columns > 0 ? process.stdout.columns : 80,
        rows: process.stdout.rows > 0 ? process.stdout.rows : 24,
      };
    }
    return null;
  }
}

/** 进入交互模式：备用屏幕 + 隐藏光标 + raw input。 */
export function enterInteractive(tty: Tty): void {
  tty.read.setRawMode(true);
  tty.write.write('\x1b[?1049h\x1b[?25l\x1b[2J\x1b[H');
}

/** 退出交互模式：恢复原始终端状态。 */
export function exitInteractive(tty: Tty): void {
  tty.write.write('\x1b[?25h\x1b[?1049l');
  tty.read.setRawMode(false);
}

/** 全量重绘一帧画面。 */
export function renderFrame(tty: Tty, frame: string): void {
  tty.write.write('\x1b[2J\x1b[H' + frame);
}

/** 单行确认询问（auto 首次确认）：canonical 输入整行 y/N，超时默认拒绝。 */
export function promptConfirm(
  tty: Tty,
  text: string,
  timeoutMs = 30000,
): Promise<string | null> {
  tty.write.write(text);
  return new Promise((resolve) => {
    let buf = '';
    const timer = setTimeout(() => {
      cleanup();
      tty.write.write('\r\n');
      resolve(null);
    }, timeoutMs);
    const onData = (chunk: Buffer) => {
      const s = chunk.toString('utf8');
      for (const ch of s) {
        if (ch === '\r' || ch === '\n') {
          cleanup();
          tty.write.write('\r\n');
          resolve(buf);
          return;
        }
        buf += ch;
      }
    };
    const cleanup = () => {
      clearTimeout(timer);
      tty.read.removeListener('data', onData);
    };
    tty.read.on('data', onData);
  });
}

/**
 * KeyReader — raw 输入流 → 按键序列解析。
 *
 * 支持：ASCII 字符、UTF-8 多字节字符、回车/退格/控制键、CSI 序列
 * （\x1b[A 上、\x1b[B 下、\x1b[C 右、\x1b[D 左）。
 */
export class KeyReader {
  private decoder = new StringDecoder('utf8');
  private buf = '';
  private waiters: Array<(k: Key | null) => void> = [];
  private closed = false;

  constructor(readStream: ReadStream) {
    readStream.on('data', (chunk: Buffer) => {
      this.buf += this.decoder.write(chunk);
      this.pump();
    });
    readStream.on('end', () => this.pumpEnd());
    readStream.on('error', () => this.pumpEnd());
  }

  /** 取下一个按键；流关闭返回 null。 */
  next(): Promise<Key | null> {
    if (this.buf !== '') {
      const k = consumeKey(this.buf);
      if (k !== null) {
        this.buf = this.buf.slice(k.len);
        return Promise.resolve(k.key);
      }
    }
    if (this.closed) {
      return Promise.resolve(null);
    }
    return new Promise((resolve) => this.waiters.push(resolve));
  }

  private pumpEnd(): void {
    this.closed = true;
    while (this.waiters.length > 0) {
      this.waiters.shift()!(null);
    }
  }

  private pump(): void {
    while (this.waiters.length > 0 && this.buf !== '') {
      const k = consumeKey(this.buf);
      if (k === null) {
        return; // 等待更多字节
      }
      this.buf = this.buf.slice(k.len);
      this.waiters.shift()!(k.key);
    }
  }
}

interface Consumed {
  key: Key;
  len: number;
}

/** 从输入缓冲开头消费一个按键；字节不足返回 null（等待更多输入）。 */
function consumeKey(buf: string): Consumed | null {
  if (buf === '') {
    return null;
  }
  const c = buf[0];

  // 控制字符
  switch (c) {
    case '\r':
      return { key: 'enter', len: 1 };
    case '\t':
      return { key: 'tab', len: 1 };
    case '\x7f':
    case '\x08':
      return { key: 'backspace', len: 1 };
    case '\x03':
      return { key: 'ctrl+c', len: 1 };
    case '\x10':
      return { key: 'ctrl+p', len: 1 };
    case '\x0e':
      return { key: 'ctrl+n', len: 1 };
    case '\x15':
      return { key: 'ctrl+u', len: 1 };
    case '\x17':
      return { key: 'ctrl+w', len: 1 };
    case ' ':
      return { key: 'space', len: 1 };
  }

  // CSI 序列：\x1b[A / \x1b[B / \x1b[C / \x1b[D
  if (c === '\x1b') {
    if (buf.length === 1) {
      return null; // 等待后续字节确认是否为序列开头
    }
    const d = buf[1];
    if (d === '[') {
      if (buf.length < 3) {
        return null;
      }
      switch (buf[2]) {
        case 'A':
          return { key: 'up', len: 3 };
        case 'B':
          return { key: 'down', len: 3 };
        case 'C':
        case 'D':
          return { key: { type: 'char', value: '' }, len: 3 }; // 左右键忽略
        default:
          return { key: 'esc', len: 3 };
      }
    }
    return { key: 'esc', len: 1 };
  }

  // 可打印字符：单字节 ASCII 或 UTF-8 多字节
  if (c.charCodeAt(0) < 0x20) {
    return { key: { type: 'char', value: c }, len: 1 }; // 其余控制键忽略
  }
  const seqLen = utf8SeqLen(buf);
  if (seqLen > buf.length) {
    return null; // 多字节字符尚未收全
  }
  return { key: { type: 'char', value: buf.slice(0, seqLen) }, len: seqLen };
}

/** 计算 buf 起始 UTF-8 字符的字节长度（按首字节高 2 位判断）。 */
function utf8SeqLen(buf: string): number {
  const b = buf.charCodeAt(0);
  if (b < 0x80) {
    return 1;
  }
  if ((b & 0xe0) === 0xc0) {
    return 2;
  }
  if ((b & 0xf0) === 0xe0) {
    return 3;
  }
  if ((b & 0xf8) === 0xf0) {
    return 4;
  }
  return 1;
}