import { WorkflowDocument } from '@flowgram.ai/free-layout-editor';

/** Agent 节点中真正需要校验必填的字段，其余 required 字段忽略 */
const AGENT_HARNESS_REQUIRED_KEYS = new Set(['llmConfigId', 'llmModelEntryId']);

export async function collectWorkflowProblems(
  document: WorkflowDocument
): Promise<Array<{ nodeId: string; title: string; messages?: string[] }>> {
  const nodes = document.getAllNodes();
  const problemsMap = new Map<string, string[]>();

  // 第1道：form.validate() 校验
  await Promise.all(nodes.map(async (n) => n.form?.validate()));
  for (const n of nodes.filter((n) => n.form?.state.invalid)) {
    const json: any = document.toNodeJSON(n);
    const title = json?.data?.title;
    const key = title ? String(title) : n.id;
    // 从 form 的 feedback 中收集具体错误信息
    const feedbacks = (n.form?.state as any)?.feedbacks;
    const msgs: string[] = [];
    if (feedbacks && typeof feedbacks === 'object') {
      for (const [, fb] of Object.entries(feedbacks as Record<string, any>)) {
        if (fb?.message) msgs.push(fb.message);
      }
    }
    problemsMap.set(n.id, msgs.length > 0 ? msgs : [key]);
  }

  // 第2道：手动遍历 required 字段校验
  const toJSONList = nodes.map((n) => document.toNodeJSON(n));
  const getVal = (v: any) => {
    if (!v) return undefined;
    if (typeof v.content !== 'undefined') return v.content;
    return v;
  };
  const isEmpty = (schema: any, val: any) => {
    const t = schema?.type;
    if (t === 'string') return !(typeof val === 'string' && val.trim().length > 0);
    if (t === 'number') return !(typeof val === 'number');
    if (t === 'boolean') return typeof val === 'boolean' ? false : true;
    if (t === 'array') return Array.isArray(val) ? false : true;
    if (t === 'object') return typeof val === 'object' && val !== null ? false : true;
    return val === undefined || val === null;
  };
  const validateNode = (json: any) => {
    const inputs = json?.data?.inputs;
    const values = json?.data?.inputsValues;
    const nodeType: string = json?.type || json?.data?.type || '';
    const requiredKeys: string[] = Array.isArray(inputs?.required) ? inputs.required : [];
    for (const k of requiredKeys) {
      // Agent 节点：只校验关键字段
      if (nodeType === 'ai/agentHarness' && !AGENT_HARNESS_REQUIRED_KEYS.has(k)) {
        continue;
      }
      const schema = inputs?.properties?.[k];
      const v = getVal(values?.[k]);
      if (isEmpty(schema, v)) {
        const id = String(json?.id);
        const label = schema?.extra?.label || k;
        const existing = problemsMap.get(id) || [];
        if (!existing.includes(label)) {
          problemsMap.set(id, [...existing, `${label} is required`]);
        }
      }
    }
    const blocks = Array.isArray(json?.blocks) ? json.blocks : [];
    for (const b of blocks) validateNode(b);
  };
  for (const j of toJSONList) validateNode(j);

  return Array.from(problemsMap.entries()).map(([nodeId, messages]) => ({
    nodeId,
    title: messages[0],
    messages,
  }));
}
