export type RuleChainParamType = "string" | "number" | "boolean" | "array" | "object";

export type RuleChainParamNode = {
  id: string;
  key: string;
  type: RuleChainParamType;
  required: boolean;
  description: string;
  children: RuleChainParamNode[];
};

type StoredParamNode = {
  key?: string;
  type?: string;
  value?: unknown;
  required?: boolean;
  description?: string;
  children?: unknown;
};

const PARAM_TYPES: RuleChainParamType[] = ["string", "number", "boolean", "array", "object"];

function isParamType(s: string): s is RuleChainParamType {
  return (PARAM_TYPES as string[]).includes(s);
}

export function newParamNodeId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `r_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}

export function emptyRuleChainParamsJson(): string {
  return "[]";
}

function inferType(value: unknown): RuleChainParamType {
  if (typeof value === "number" && Number.isFinite(value)) return "number";
  if (typeof value === "boolean") return "boolean";
  if (Array.isArray(value)) return "array";
  if (value && typeof value === "object") return "object";
  return "string";
}

function parseNode(raw: unknown): RuleChainParamNode | null {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return null;
  const o = raw as StoredParamNode;
  const typ = typeof o.type === "string" && isParamType(o.type) ? o.type : inferType(o.value);
  const childrenRaw = Array.isArray(o.children) ? o.children : [];
  return {
    id: newParamNodeId(),
    key: typeof o.key === "string" ? o.key : "",
    type: typ,
    required: Boolean(o.required),
    description: typeof o.description === "string" ? o.description : "",
    children: childrenRaw.map(parseNode).filter((n): n is RuleChainParamNode => Boolean(n)),
  };
}

export function parseRuleChainParamsJson(json: string): RuleChainParamNode[] {
  const raw = json?.trim() || "[]";
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return [];
  }
  if (Array.isArray(parsed)) {
    return parsed.map(parseNode).filter((n): n is RuleChainParamNode => Boolean(n));
  }
  // 兼容直接粘贴对象 JSON：将其转换成顶层节点
  if (parsed && typeof parsed === "object") {
    return importRuleChainParamsFromObjectJson(JSON.stringify(parsed));
  }
  return [];
}

function serializeNode(node: RuleChainParamNode): Record<string, unknown> {
  return {
    key: node.key.trim(),
    type: node.type,
    required: node.required,
    description: node.description.trim(),
    children: node.children.map(serializeNode),
  };
}

export function serializeRuleChainParamsNodes(nodes: RuleChainParamNode[]): string {
  const items = nodes
    .filter((n) => n.key.trim() !== "")
    .map(serializeNode);
  return JSON.stringify(items, null, 2);
}

/** 将已存储的 JSON 字符串格式化为缩进展示；无法解析时原样返回 */
export function formatRuleChainParamsJsonPretty(raw: string): string {
  const t = raw?.trim() || "[]";
  try {
    return JSON.stringify(JSON.parse(t), null, 2);
  } catch {
    return raw;
  }
}

/**
 * 校验并解析参数树 JSON（数组），失败时抛出带中文说明的错误。
 */
export function parseRuleChainParamsJsonStrict(raw: string): RuleChainParamNode[] {
  const trimmed = raw?.trim() || "[]";
  let parsed: unknown;
  try {
    parsed = JSON.parse(trimmed);
  } catch (e) {
    throw new Error(`JSON 语法错误：${e instanceof Error ? e.message : String(e)}`);
  }
  if (!Array.isArray(parsed)) {
    throw new Error("顶层必须是 JSON 数组");
  }
  for (let i = 0; i < parsed.length; i++) {
    const item = parsed[i];
    if (!item || typeof item !== "object" || Array.isArray(item)) {
      throw new Error(`第 ${i + 1} 项必须是对象`);
    }
    const o = item as StoredParamNode;
    if (typeof o.key !== "string") {
      throw new Error(`第 ${i + 1} 项须包含字符串字段 key`);
    }
    if (o.type !== undefined && typeof o.type === "string" && o.type !== "" && !isParamType(o.type)) {
      throw new Error(`第 ${i + 1} 项 type 须为 string、number、boolean、array、object 之一`);
    }
  }
  return parseRuleChainParamsJson(trimmed);
}

/** 去掉 JSON 字符串外的 // 行注释与 /* *\/ 块注释，便于粘贴带说明的 JSON。 */
function stripJsonLikeComments(input: string): string {
  let out = "";
  let i = 0;
  let inString = false;
  let escape = false;
  while (i < input.length) {
    const c = input[i]!;
    if (inString) {
      out += c;
      if (escape) {
        escape = false;
      } else if (c === "\\") {
        escape = true;
      } else if (c === '"') {
        inString = false;
      }
      i++;
      continue;
    }
    if (c === '"') {
      inString = true;
      out += c;
      i++;
      continue;
    }
    if (c === "/" && input[i + 1] === "/") {
      i += 2;
      while (i < input.length && input[i] !== "\n" && input[i] !== "\r") {
        i++;
      }
      continue;
    }
    if (c === "/" && input[i + 1] === "*") {
      i += 2;
      while (i < input.length - 1) {
        if (input[i] === "*" && input[i + 1] === "/") {
          i += 2;
          break;
        }
        i++;
      }
      continue;
    }
    out += c;
    i++;
  }
  return out;
}

function isPlainObject(v: unknown): v is Record<string, unknown> {
  return Boolean(v && typeof v === "object" && !Array.isArray(v));
}

/** 将数组中的多个对象合并为一份键 → 示例值，用于推导数组元素的对象结构。 */
function mergeObjectArrayItemsForSchema(items: unknown[]): Record<string, unknown> {
  const merged: Record<string, unknown> = {};
  for (const el of items) {
    if (!isPlainObject(el)) continue;
    for (const [k, v] of Object.entries(el)) {
      if (!(k in merged)) merged[k] = v;
    }
  }
  return merged;
}

function valueToParamNode(key: string, val: unknown): RuleChainParamNode {
  const typ = inferType(val);
  if (typ === "object" && isPlainObject(val)) {
    return {
      id: newParamNodeId(),
      key,
      type: "object",
      required: false,
      description: "",
      children: objectToNodes(val),
    };
  }
  if (typ === "array" && Array.isArray(val)) {
    const arr = val as unknown[];
    if (arr.length === 0) {
      return {
        id: newParamNodeId(),
        key,
        type: "array",
        required: false,
        description: "",
        children: [],
      };
    }
    const first = arr[0];
    if (isPlainObject(first)) {
      const itemFields = mergeObjectArrayItemsForSchema(arr);
      return {
        id: newParamNodeId(),
        key,
        type: "array",
        required: false,
        description: "",
        children: [
          {
            id: newParamNodeId(),
            key: "",
            type: "object",
            required: false,
            description: "",
            children: objectToNodes(itemFields),
          },
        ],
      };
    }
    const itemTyp = inferType(first);
    return {
      id: newParamNodeId(),
      key,
      type: "array",
      required: false,
      description: "",
      children: [
        {
          id: newParamNodeId(),
          key: "",
          type: itemTyp,
          required: false,
          description: "",
          children: [],
        },
      ],
    };
  }
  return {
    id: newParamNodeId(),
    key,
    type: typ,
    required: false,
    description: "",
    children: [],
  };
}

function objectToNodes(obj: Record<string, unknown>): RuleChainParamNode[] {
  return Object.entries(obj).map(([k, val]) => valueToParamNode(k, val));
}

export function importRuleChainParamsFromObjectJson(text: string): RuleChainParamNode[] {
  const trimmed = text.trim();
  if (!trimmed) {
    throw new Error("JSON 内容为空");
  }
  const withoutComments = stripJsonLikeComments(trimmed);
  let parsed: unknown;
  try {
    parsed = JSON.parse(withoutComments);
  } catch (e) {
    throw new Error(`JSON 解析失败：${e instanceof Error ? e.message : String(e)}`);
  }
  if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
    throw new Error("JSON 须为对象：{ \"参数名\": 参数值, ... }");
  }
  return objectToNodes(parsed as Record<string, unknown>);
}

function defaultValueForLeafType(typ: RuleChainParamType): unknown {
  if (typ === "number") return 0;
  if (typ === "boolean") return false;
  if (typ === "array") return [];
  if (typ === "object") return {};
  return "";
}

function nodeToJsonValue(node: RuleChainParamNode): unknown {
  if (node.type === "object") {
    return buildRuleChainParamsPreviewValue(node.children);
  }
  if (node.type === "array") {
    if (node.children.length > 0) {
      return [nodeToJsonValue(node.children[0])];
    }
    return [];
  }
  return defaultValueForLeafType(node.type);
}

/** 与预览生成一致：单节点对应的示例 JSON 值（供执行规则等表单初始化） */
export function paramNodeToSampleJsonValue(node: RuleChainParamNode): unknown {
  return nodeToJsonValue(node);
}

export function buildRuleChainParamsPreviewValue(nodes: RuleChainParamNode[]): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const node of nodes) {
    const key = node.key.trim();
    if (!key) continue;
    out[key] = nodeToJsonValue(node);
  }
  return out;
}

function previewLiteral(node: RuleChainParamNode): string {
  const value = nodeToJsonValue(node);
  if (typeof value === "string") return JSON.stringify(value);
  return JSON.stringify(value);
}

function buildCommentedLines(nodes: RuleChainParamNode[], indent = 0): string[] {
  const pad = "  ".repeat(indent);
  const lines: string[] = [];
  nodes.forEach((node, idx) => {
    const key = node.key.trim();
    if (!key) return;
    const comma = idx === nodes.length - 1 ? "" : ",";
    const comment = node.description.trim() ? ` // ${node.description.trim()}` : "";
    if (node.type === "object") {
      lines.push(`${pad}"${key}": {${comment}`);
      lines.push(...buildCommentedLines(node.children, indent + 1));
      lines.push(`${pad}}${comma}`);
      return;
    }
    if (node.type === "array" && node.children.length > 0) {
      lines.push(`${pad}"${key}": [${comment}`);
      const child = node.children[0];
      if (child.type === "object") {
        lines.push(`${pad}  {`);
        lines.push(...buildCommentedLines(child.children, indent + 2));
        lines.push(`${pad}  }`);
      } else {
        lines.push(`${pad}  ${previewLiteral(child)}`);
      }
      lines.push(`${pad}]${comma}`);
      return;
    }
    lines.push(`${pad}"${key}": ${previewLiteral(node)}${comma}${comment}`);
  });
  return lines;
}

export function buildRuleChainParamsCommentedPreview(nodes: RuleChainParamNode[]): string {
  const lines = ["{", ...buildCommentedLines(nodes, 1), "}"];
  return lines.join("\n");
}
