/**
 * toml — 极简 TOML 子集解析/编码（仅覆盖 history.toml / confirm.toml 所需结构）。
 *
 * 兼容 Go 引擎（BurntSushi/toml）写出的文件：
 *   - 注释行（# …）、空行
 *   - 表头 `[table]` / `["quoted key"]`
 *   - `key = "string"`（含基本字符串转义 \n \t \" \\ \r \b \f \uXXXX）
 *   - `"label" = ["a", "b"]`（字符串数组）
 * 值仅支持 string / string[]——本包状态文件所需的最小类型集。
 */
export type TomlValue = string | string[];

/** 解析结果：表名 → { key: 值 }（保持文件内出现顺序）。 */
export type TomlDoc = Map<string, Map<string, TomlValue>>;

/** 裸 key 允许的字符（对齐 TOML bare key；其余 key 编码时加引号）。 */
const BARE_KEY_RE = /^[A-Za-z0-9_-]+$/;

/** 转义字符串内容（写入值时用）。 */
export function escapeTomlString(s: string): string {
  let out = '';
  for (const ch of s) {
    switch (ch) {
      case '\\':
        out += '\\\\';
        break;
      case '"':
        out += '\\"';
        break;
      case '\n':
        out += '\\n';
        break;
      case '\t':
        out += '\\t';
        break;
      case '\r':
        out += '\\r';
        break;
      case '\b':
        out += '\\b';
        break;
      case '\f':
        out += '\\f';
        break;
      default:
        out += ch;
    }
  }
  return out;
}

/** 带引号的字符串字面量（含转义）。 */
export function quoteTomlString(s: string): string {
  return '"' + escapeTomlString(s) + '"';
}

/** 表头/键名编码：裸 key 直接输出，否则带引号。 */
export function keyLiteral(key: string): string {
  return BARE_KEY_RE.test(key) ? key : quoteTomlString(key);
}

/**
 * 解析 TOML 子集。无法解析的行跳过（容错：状态文件损坏不阻断主流程）。
 * 无任何可解析内容时返回 null。
 */
export function parseToml(text: string): TomlDoc | null {
  const doc: TomlDoc = new Map();
  let cur: Map<string, TomlValue> | null = null;
  let sawContent = false;

  for (const rawLine of text.split('\n')) {
    const line = rawLine.trim();
    if (line === '' || line.startsWith('#')) {
      continue;
    }
    if (line.startsWith('[') && line.endsWith(']')) {
      const name = parseQuotedOrBare(line.slice(1, -1).trim());
      if (name === null) {
        continue;
      }
      cur = new Map();
      doc.set(name, cur);
      sawContent = true;
      continue;
    }
    if (cur === null) {
      continue; // 表头之前的游离键：本包格式不存在，跳过
    }
    const eq = line.indexOf('=');
    if (eq < 0) {
      continue;
    }
    const keyRaw = line.slice(0, eq).trim();
    const key = parseQuotedOrBare(keyRaw);
    if (key === null) {
      continue;
    }
    const valueRaw = line.slice(eq + 1).trim();
    const value = parseTomlValue(valueRaw);
    if (value === null) {
      continue;
    }
    cur.set(key, value);
    sawContent = true;
  }
  if (!sawContent) {
    return null;
  }
  return doc;
}

/** 解析 `"quoted"` 或裸字符串；失败返回 null。 */
function parseQuotedOrBare(s: string): string | null {
  if (s.startsWith('"')) {
    return parseTomlString(s);
  }
  return s;
}

/** 解析一个 TOML 字符串字面量（从首个引号到闭合引号，含转义）。 */
function parseTomlString(s: string): string | null {
  if (!s.startsWith('"')) {
    return null;
  }
  let out = '';
  let i = 1;
  while (i < s.length) {
    const ch = s[i];
    if (ch === '"') {
      return out;
    }
    if (ch === '\\') {
      const esc = s[i + 1];
      if (esc === undefined) {
        return null;
      }
      switch (esc) {
        case 'n':
          out += '\n';
          break;
        case 't':
          out += '\t';
          break;
        case 'r':
          out += '\r';
          break;
        case 'b':
          out += '\b';
          break;
        case 'f':
          out += '\f';
          break;
        case '"':
          out += '"';
          break;
        case '\\':
          out += '\\';
          break;
        case 'u': {
          const hex = s.slice(i + 2, i + 6);
          if (!/^[0-9a-fA-F]{4}$/.test(hex)) {
            return null;
          }
          out += String.fromCodePoint(parseInt(hex, 16));
          i += 4;
          break;
        }
        default:
          return null; // 不识别的转义
      }
      i += 2;
      continue;
    }
    // TOML 基本字符串不允许裸控制字符
    if (ch < ' ' && ch !== '\t') {
      return null;
    }
    out += ch;
    i++;
  }
  return null; // 未闭合
}

/** 解析字符串或字符串数组字面量；失败返回 null。 */
function parseTomlValue(raw: string): TomlValue | null {
  if (raw.startsWith('[')) {
    if (!raw.endsWith(']')) {
      return null;
    }
    const inner = raw.slice(1, -1).trim();
    if (inner === '') {
      return [];
    }
    const values: string[] = [];
    let i = 0;
    while (i < inner.length) {
      const ch = inner[i];
      if (ch === ',' || ch === ' ' || ch === '\t') {
        i++;
        continue;
      }
      if (ch === '"') {
        const start = i;
        let j = i + 1;
        let escaped = false;
        while (j < inner.length) {
          const c = inner[j];
          if (c === '"' && !escaped) {
            break;
          }
          if (c === '\\' && !escaped) {
            escaped = true;
          } else {
            escaped = false;
          }
          j++;
        }
        if (j >= inner.length) {
          return null;
        }
        const literal = parseTomlString(inner.slice(start, j + 1));
        if (literal === null) {
          return null;
        }
        values.push(literal);
        i = j + 1;
      } else {
        return null; // 仅支持字符串数组
      }
    }
    return values;
  }
  if (raw.startsWith('"')) {
    return parseTomlString(raw);
  }
  return null; // 布尔/数字等非本包类型
}