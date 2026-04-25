/**
 * 节点级 DSL 特例转换（messages/params 等），与 specs 中的 NodeMappingSpec 配合使用。
 */

/** 与 rulechain-builder 中 restApiCall 导出一致：params 对象序列化为 query 段（不含前导 ?/&）。 */
function serializeParamsToQuery(params: Record<string, unknown>): string {
  return Object.entries(params)
    .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v ?? ''))}`)
    .join('&');
}

/** 解析 query 段为键值（与 buildDocumentFromRuleChainJSON 中 split/decode 行为一致）。 */
function parseQueryStringPairs(qs: string): Record<string, string> {
  const queryValues: Record<string, string> = {};
  for (const pair of qs.split('&')) {
    if (!pair) continue;
    const eq = pair.indexOf('=');
    const rawK = eq >= 0 ? pair.slice(0, eq) : pair;
    const rawV = eq >= 0 ? pair.slice(eq + 1) : '';
    if (!rawK) continue;
    const k = decodeURIComponent(rawK);
    const v = decodeURIComponent(rawV);
    queryValues[k] = v;
  }
  return queryValues;
}

/**
 * toDSL：在 restUrlBase 上拼接 params 的 query，得到 restEndpointUrlPattern（与历史 rulechain-builder
 * restApiCall 分支一致：pattern 已有 `?` 时用 `&` 追加）。
 */
export function transformRestApiCallOut(config: Record<string, unknown>): Record<string, unknown> {
  const restUrlBase = String(config.restUrlBase ?? '');
  const params =
    config.params && typeof config.params === 'object' && !Array.isArray(config.params)
      ? (config.params as Record<string, unknown>)
      : {};
  const headers =
    config.headers && typeof config.headers === 'object' && !Array.isArray(config.headers)
      ? (config.headers as Record<string, unknown>)
      : {};

  // 与历史 rulechain-builder 一致：仅当存在至少一个键时才写入 params / headers，避免导出 `{}` 占位。
  const out: Record<string, unknown> = {
    requestMethod: config.requestMethod,
  };
  if (Object.keys(params).length > 0) {
    out.params = params;
  }
  if (Object.keys(headers).length > 0) {
    out.headers = headers;
  }
  if (config.body !== undefined && config.body !== null && String(config.body).trim() !== '') {
    out.body = config.body;
  }
  if (config.readTimeoutMs !== undefined && config.readTimeoutMs !== null) {
    out.readTimeoutMs = config.readTimeoutMs;
  }

  // 与历史 rulechain-builder 一致：仅当画布 URL 非空时才写入 restEndpointUrlPattern；无基址时仍可单独导出 params。
  if (restUrlBase) {
    let pattern = restUrlBase;
    if (Object.keys(params).length > 0) {
      const qs = serializeParamsToQuery(params);
      const sep = pattern.includes('?') ? '&' : '?';
      pattern = qs ? `${pattern}${sep}${qs}` : pattern;
    }
    out.restEndpointUrlPattern = pattern;
  }

  return out;
}

/**
 * fromDSL：从 restEndpointUrlPattern 拆出 base URL 与 query，再与 configuration.params 合并写回 params。
 *
 * 合并顺序：先放入 URL 查询串解析结果，再以 configuration.params 覆盖（与 buildDocumentFromRuleChainJSON
 * 中 `mergedParamVals = { ...paramValsFromUrl, ...paramValsFromCfg }` 一致）——显式 DSL 里的 params 优先。
 */
export function transformRestApiCallIn(config: Record<string, unknown>): Record<string, unknown> {
  const fullUrl = String(config.restEndpointUrlPattern ?? '');
  let baseUrl = fullUrl;
  let queryValues: Record<string, string> = {};
  const qm = fullUrl.indexOf('?');
  if (qm >= 0) {
    baseUrl = fullUrl.slice(0, qm);
    queryValues = parseQueryStringPairs(fullUrl.slice(qm + 1));
  }

  const fromCfg =
    config.params && typeof config.params === 'object' && !Array.isArray(config.params)
      ? (config.params as Record<string, unknown>)
      : {};

  const mergedParams: Record<string, unknown> = { ...queryValues, ...fromCfg };

  return {
    ...config,
    restUrlBase: baseUrl,
    params: mergedParams,
  };
}

function formatSwitchValue(v: any): string {
  const val = v?.content;
  const isNum =
    typeof val === 'number' || (typeof val === 'string' && /^-?\d+(?:\.\d+)?$/.test(val));
  return isNum ? String(val) : JSON.stringify(String(val ?? ''));
}

function formatSwitchRow(row: any): string {
  if (!row) return '';
  if (row.content && String(row.content).trim().length > 0) {
    return String(row.content).trim();
  }
  if (row.type === 'expression') {
    const left = row.left?.content ?? '';
    const op = row.operator ?? '';
    const right = formatSwitchValue(row.right ?? {});
    if (left && op && right) return `${left} ${op} ${right}`;
  }
  return '';
}

function formatSwitchGroup(g: any): string {
  const rows = Array.isArray(g?.rows) ? g.rows : [];
  const exprs = rows.map(formatSwitchRow).filter((s: string) => s && s.length > 0);
  const joiner = g?.operator === 'or' ? ' || ' : ' && ';
  if (exprs.length === 0) return '';
  const joined = exprs.join(joiner);
  return exprs.length > 1 ? `(${joined})` : joined;
}

function splitTopLevel(expr: string, delim: '||' | '&&'): string[] {
  const parts: string[] = [];
  let buf = '';
  let depth = 0;
  let inSingle = false;
  let inDouble = false;
  let inTemplate = false;
  for (let i = 0; i < expr.length; i++) {
    const ch = expr[i];
    const prev = expr[i - 1];
    if (!inSingle && !inDouble && !inTemplate) {
      if (ch === '(') depth++;
      else if (ch === ')') depth = Math.max(0, depth - 1);
      else if (ch === "'" && prev !== '\\') inSingle = true;
      else if (ch === '"' && prev !== '\\') inDouble = true;
      else if (ch === '`' && prev !== '\\') inTemplate = true;
      const isDelim =
        delim === '||' ? expr.slice(i, i + 2) === '||' : expr.slice(i, i + 2) === '&&';
      if (isDelim && depth === 0) {
        parts.push(buf.trim());
        buf = '';
        i++;
        continue;
      }
    } else {
      if (inSingle && ch === "'" && prev !== '\\') inSingle = false;
      else if (inDouble && ch === '"' && prev !== '\\') inDouble = false;
      else if (inTemplate && ch === '`' && prev !== '\\') inTemplate = false;
    }
    buf += ch;
  }
  if (buf.trim()) parts.push(buf.trim());
  return parts.filter((p) => p.length > 0);
}

function parseSwitchRow(rowExpr: string): any {
  const expr = rowExpr.trim();
  const ops = ['contains', '==', '!=', '>=', '<=', '>', '<'];
  let foundOp = '';
  let left = '';
  let right = '';
  for (const op of ops) {
    const idx = expr.indexOf(op);
    if (idx > 0) {
      foundOp = op;
      left = expr.slice(0, idx).trim();
      right = expr.slice(idx + op.length).trim();
      break;
    }
  }
  if (!foundOp) return { type: 'expression', content: expr };
  const stripQuotes = (s: string) => {
    const t = s.trim();
    if ((t.startsWith("'") && t.endsWith("'")) || (t.startsWith('"') && t.endsWith('"'))) {
      return t.slice(1, -1);
    }
    return t;
  };
  return {
    type: 'expression',
    content: '',
    left: { type: 'constant', content: left },
    operator: foundOp,
    right: { type: 'constant', content: stripQuotes(right) },
  };
}

/** toDSL：switch 的可视化 cases -> DSL expressions。 */
export function transformSwitchConfigOut(config: Record<string, unknown>): Record<string, unknown> {
  const rawCases = Array.isArray(config.cases) ? config.cases : [];
  const cases = rawCases
    .map((c: any) => {
      const groups = Array.isArray(c?.groups) ? c.groups : [];
      const groupExprs = groups.map(formatSwitchGroup).filter((s: string) => s && s.length > 0);
      const fullExpr = groupExprs.join(' || ');
      return { case: fullExpr, then: String(c?.key ?? '') };
    })
    .filter((item) => item.case && item.then);
  return { cases };
}

/** fromDSL：switch DSL expressions -> 可视化 cases。 */
export function transformSwitchConfigIn(config: Record<string, unknown>): Record<string, unknown> {
  const rawCases = Array.isArray(config.cases) ? config.cases : [];
  const cases = rawCases.map((c: any) => {
    const expr = String(c?.case ?? '');
    const groupsExpr = splitTopLevel(expr, '||');
    const groups = groupsExpr.map((ge) => {
      const rowsExpr = splitTopLevel(ge, '&&');
      return { operator: 'and', rows: rowsExpr.map(parseSwitchRow) };
    });
    return { key: String(c?.then ?? ''), groups };
  });
  return { ...config, cases };
}

/** fromDSL：历史 cursorPath 回填为 agentPath（官方 CLI 可执行文件为 agent，参见 Cursor CLI 文档）。 */
export function transformCursorCliConfigIn(
  config: Record<string, unknown>
): Record<string, unknown> {
  const next = { ...config };
  const ap = next.agentPath;
  const emptyAp =
    ap === undefined || ap === null || (typeof ap === 'string' && String(ap).trim() === '');
  if (emptyAp && next.cursorPath != null && String(next.cursorPath).trim() !== '') {
    next.agentPath = next.cursorPath;
  }
  return next;
}

/** fromDSL：agentHarness Skill 白名单历史为逗号分隔字符串时转为 string[]，便于 JSON 数组 DSL。 */
export function transformAgentHarnessConfigIn(
  config: Record<string, unknown>
): Record<string, unknown> {
  const next = { ...config };
  // Workspace 工具在 Agent-LLM 节点中固定开启，不允许关闭。
  next.enableWorkspaceTools = true;
  // 默认启用子 Agent 工具；旧 DSL 缺失该字段时回填 true。
  const sub = next.enableSubAgentTool;
  if (sub === undefined || sub === null || String(sub).trim() === '') {
    next.enableSubAgentTool = true;
  }
  const skill = next.skillAllowlist;
  if (typeof skill === 'string') {
    const t = skill.trim();
    next.skillAllowlist =
      t === ''
        ? []
        : t
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean);
  }
  return next;
}

/** fromDSL：旧规则链无 msgType/postLang 时回填为 text/zh_cn；补全 post/@ 与卡片预设字段。 */
export function transformFeishuWebhookConfigIn(
  config: Record<string, unknown>
): Record<string, unknown> {
  const next = { ...config };
  const mt = next.msgType;
  if (mt === undefined || mt === null || String(mt).trim() === '') {
    next.msgType = 'text';
  }
  const pl = next.postLang;
  if (pl === undefined || pl === null || String(pl).trim() === '') {
    next.postLang = 'zh_cn';
  }
  if (!Array.isArray(next.postMentionUserIds)) {
    next.postMentionUserIds = [];
  }
  const ip = next.interactivePreset;
  if (ip === undefined || ip === null || String(ip).trim() === '') {
    next.interactivePreset = 'card_json';
  }
  return next;
}
