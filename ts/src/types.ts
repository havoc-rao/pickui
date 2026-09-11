/**
 * types — 与 docs/integration/protocol.md 对齐的纯类型（零依赖）。
 *
 * 字段名/形状是协议 v1 的一部分，任何改动需同步 protocol.md 与 Go 引擎。
 */

/** 结构化候选：value 为选中值，desc 仅供展示（来自 key<TAB>description 拆分）。 */
export interface Candidate {
  value: string;
  desc?: string;
}

/** 过滤模式（对齐引擎 --mode）。 */
export type FilterMode = 'substring' | 'token' | 'fuzzy';

/** 过滤选项（对齐引擎 filter --json 的 --mode/--sep）。 */
export interface FilterOptions {
  mode?: FilterMode;
  /** 仅 token 模式：分隔符集合，默认 "_"，可多字符如 ":_"。 */
  sep?: string;
}

/** filter 协议输出：命中候选 + 高亮 rune 区间 [start, end)。 */
export interface FilteredCandidate {
  value: string;
  desc: string;
  /** 无 query 时恒为 []（非 null）。 */
  ranges: [number, number][];
}

/**
 * pick 命令透传 flags（对齐协议 pick 子命令）。
 * 注意：--from/--map 属于引擎侧候选来源，绑定层不额外包装——需要时请用
 * rawPick([...args]) 或直接调用引擎二进制。
 */
export interface PickFlags {
  /** 预填查询；非交互时过滤后取首个匹配。 */
  query?: string;
  /** token 前缀模式分隔符集合（默认 "_"）。 */
  sep?: string;
  /** 子序列模糊匹配。 */
  fuzzy?: boolean;
  /** 选择记忆键（history.toml）。 */
  label?: string;
  /** 仅一个匹配时自动选中。 */
  select1?: boolean;
  /** 配合 query：精确或唯一前缀自动选中（首次需引擎侧确认）。 */
  auto?: boolean;
}

/** 引擎调用失败（退出码非预期）时抛出的错误。 */
export class PicktuiError extends Error {
  readonly exitCode: number;
  readonly stderr: string;

  constructor(message: string, exitCode: number, stderr: string) {
    super(message);
    this.name = 'PicktuiError';
    this.exitCode = exitCode;
    this.stderr = stderr;
  }
}

/** resolve 协议输出。 */
export interface ResolveResult {
  value: string;
}

/** 引擎 version 输出（"picktui <semver>"）。 */
export interface EngineVersion {
  /** 完整输出行，如 "picktui 0.1.0"。 */
  raw: string;
  /** 语义化版本号，如 "0.1.0"。 */
  version: string;
}