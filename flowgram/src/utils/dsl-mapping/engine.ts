import type { MappingValueType, NodeLike, NodeMappingSpec, FlowValueLike } from './types';

export type DslConfiguration = Record<string, unknown>;

/** 写入 inputsValues 的最小结构（与 IFlowValue 的 content 对齐）。 */
export type InputsValuesMap = Record<string, { content: unknown }>;

/**
 * Spec 默认值语义（单一事实源在 `specs.ts` 的 `defaultValue`）：
 *
 * - **toDSL**：`inputsValues` 缺键，或内容被 `isEmptyForMapping` 判空时，使用该字段的 `MappingField.defaultValue`
 *  （见 `normalizeOut`）。导出结果以 spec 为准，不与历史「隐式缺省」混用。
 * - **fromDSL**：`configuration` 某键为 `undefined` 时用 `defaultValue`；显式 `null` 保留为 `null`（见 `normalizeIn`）。
 *
 * `transformOut` / `transformIn` 抛错时：`console.warn` 后回退到钩子执行前的值，不阻断整次映射。
 */

function isEmptyForMapping(raw: unknown, valueType: MappingValueType): boolean {
  if (raw === undefined || raw === null) {
    return true;
  }
  if (valueType === 'boolean' && typeof raw === 'string' && raw.trim() === '') {
    return true;
  }
  if (valueType === 'template' || valueType === 'constant' || valueType === 'json') {
    if (typeof raw === 'string' && raw.trim() === '') {
      return true;
    }
  }
  return false;
}

function normalizeBoolean(raw: unknown): boolean | undefined {
  if (raw === true || raw === false) {
    return raw;
  }
  if (typeof raw === 'number') {
    if (raw === 1) {
      return true;
    }
    if (raw === 0) {
      return false;
    }
    return undefined;
  }
  if (typeof raw === 'string') {
    const s = raw.trim().toLowerCase();
    if (s === 'true' || s === '1' || s === 'yes') {
      return true;
    }
    if (s === 'false' || s === '0' || s === 'no') {
      return false;
    }
  }
  return undefined;
}

function normalizeNumber(raw: unknown): number | undefined {
  if (typeof raw === 'number' && !Number.isNaN(raw)) {
    return raw;
  }
  if (typeof raw === 'string' && raw.trim() !== '') {
    const n = Number(raw);
    if (!Number.isNaN(n)) {
      return n;
    }
  }
  return undefined;
}

function normalizeJsonOut(raw: unknown): unknown {
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (t === '') {
      return undefined;
    }
    try {
      return JSON.parse(t) as unknown;
    } catch {
      return raw;
    }
  }
  return raw;
}

/** fromDSL：与 normalizeJsonOut 对称，字符串尝试 JSON.parse，失败保留原值。 */
function normalizeJsonIn(raw: unknown): unknown {
  if (typeof raw === 'string') {
    const t = raw.trim();
    if (t === '') {
      return raw;
    }
    try {
      return JSON.parse(t) as unknown;
    } catch {
      return raw;
    }
  }
  return raw;
}

function normalizeOut(valueType: MappingValueType, raw: unknown, defaultValue: unknown): unknown {
  let v = raw;
  if (isEmptyForMapping(v, valueType)) {
    v = defaultValue;
  }
  switch (valueType) {
    case 'number': {
      const n = normalizeNumber(v);
      if (n === undefined) {
        return defaultValue !== undefined
          ? normalizeNumber(defaultValue) ?? defaultValue
          : undefined;
      }
      return n;
    }
    case 'boolean': {
      const b = normalizeBoolean(v);
      if (b === undefined) {
        return defaultValue !== undefined
          ? normalizeBoolean(defaultValue) ?? defaultValue
          : undefined;
      }
      return b;
    }
    case 'json': {
      if (isEmptyForMapping(v, valueType)) {
        return defaultValue;
      }
      return normalizeJsonOut(v);
    }
    case 'template':
    case 'constant':
    default:
      if (v === undefined || v === null) {
        return defaultValue;
      }
      return v;
  }
}

/**
 * fromDSL：仅 `undefined`（缺键）触发 defaultValue；显式 `null` 保留为 null。
 * boolean 空字符串视为 empty，与 toDSL 一致，走 defaultValue。
 */
function normalizeIn(valueType: MappingValueType, raw: unknown, defaultValue: unknown): unknown {
  if (raw === null) {
    return null;
  }

  let v: unknown = raw;
  if (v === undefined) {
    v = defaultValue;
  } else if (valueType === 'boolean' && typeof v === 'string' && v.trim() === '') {
    v = defaultValue;
  }

  switch (valueType) {
    case 'number': {
      const n = normalizeNumber(v);
      return n !== undefined ? n : defaultValue;
    }
    case 'boolean': {
      const b = normalizeBoolean(v);
      return b !== undefined ? b : defaultValue;
    }
    case 'json':
      return normalizeJsonIn(v);
    case 'template':
    case 'constant':
    default:
      return v;
  }
}

function readRawContent(
  inputsValues: Record<string, FlowValueLike | undefined> | undefined,
  inputKey: string
): unknown {
  return inputsValues?.[inputKey]?.content;
}

/**
 * 画布节点 -> DSL configuration（读取 node.data.inputsValues[key].content，默认值与类型归一化，可选 transform）。
 */
export function mapNodeToDslConfig(node: NodeLike, spec: NodeMappingSpec): DslConfiguration {
  const inputsValues = node.data?.inputsValues;
  const out: DslConfiguration = {};

  for (const field of spec.fields) {
    const rawContent = readRawContent(inputsValues, field.inputKey);
    let value = normalizeOut(field.valueType, rawContent, field.defaultValue);
    if (field.transformOut) {
      try {
        value = field.transformOut(value, { field, rawContent });
      } catch (err) {
        console.warn(
          `[dsl-mapping] field.transformOut failed nodeType=${spec.nodeType} inputKey=${field.inputKey}`,
          err
        );
      }
    }
    out[field.dslKey] = value as unknown;
  }

  if (spec.transformOut) {
    try {
      const next = spec.transformOut(out);
      return (next ?? out) as DslConfiguration;
    } catch (err) {
      console.warn(`[dsl-mapping] spec.transformOut failed nodeType=${spec.nodeType}`, err);
      return out;
    }
  }
  return out;
}

function cloneConfig(cfg: DslConfiguration): DslConfiguration {
  return { ...cfg };
}

/**
 * DSL configuration -> inputsValues 形状（供写回 node.data），支持 transformIn 与类型归一化。
 */
export function mapDslToNodeInputsValues(
  dslConfig: DslConfiguration,
  spec: NodeMappingSpec
): InputsValuesMap {
  let cfg = cloneConfig(dslConfig);
  if (spec.transformIn) {
    try {
      const next = spec.transformIn(cfg);
      cfg = (next ?? cfg) as DslConfiguration;
    } catch (err) {
      console.warn(`[dsl-mapping] spec.transformIn failed nodeType=${spec.nodeType}`, err);
    }
  }

  const inputsValues: InputsValuesMap = {};
  for (const field of spec.fields) {
    const raw = cfg[field.dslKey];
    let value = normalizeIn(field.valueType, raw, field.defaultValue);
    if (field.transformIn) {
      try {
        value = field.transformIn(value, { field });
      } catch (err) {
        console.warn(
          `[dsl-mapping] field.transformIn failed nodeType=${spec.nodeType} inputKey=${field.inputKey}`,
          err
        );
      }
    }
    inputsValues[field.inputKey] = { content: value };
  }
  return inputsValues;
}
