/**
 * DSL 映射：字段值类别（与画布表单 / DSL configuration 之间的语义对齐）。
 */
export type MappingValueType = 'template' | 'constant' | 'number' | 'boolean' | 'json';

/** 节点 data 中与 inputsValues 条目兼容的最小形状（读取 .content）。 */
export interface FlowValueLike {
  content?: unknown;
}

export interface NodeDataLike {
  inputsValues?: Record<string, FlowValueLike | undefined>;
}

export interface NodeLike {
  data?: NodeDataLike;
}

/**
 * 单字段映射：inputsValues 键 <-> DSL configuration 键。
 */
export interface MappingField {
  /** node.data.inputsValues 中的键 */
  inputKey: string;
  /** DSL configuration 中的键 */
  dslKey: string;
  valueType: MappingValueType;
  /**
   * toDSL：缺失或视为空时回填（见引擎 isEmpty 规则）。
   * fromDSL：仅 configuration 缺键（undefined）时回填；显式 `null` 保留，不由 defaultValue 替换。
   */
  defaultValue?: unknown;
  /** toDSL：归一化之后、写入 configuration 之前 */
  transformOut?: (value: unknown, ctx: { field: MappingField; rawContent: unknown }) => unknown;
  /** fromDSL：从 configuration 读出之后、写入 inputsValues 之前 */
  transformIn?: (value: unknown, ctx: { field: MappingField }) => unknown;
}

/**
 * 节点级映射规格（单一配置源的引擎输入）。
 */
export interface NodeMappingSpec {
  nodeType: string;
  fields: MappingField[];
  /** 所有字段写入后，对整份 configuration 做后处理；可返回新对象或原地修改 */
  transformOut?: (config: Record<string, unknown>) => Record<string, unknown> | void;
  /** 从 DSL 读入前，对整份 configuration 做预处理；可返回新对象或原地修改 */
  transformIn?: (config: Record<string, unknown>) => Record<string, unknown> | void;
}
