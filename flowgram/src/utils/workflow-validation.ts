import { WorkflowDocument } from '@flowgram.ai/free-layout-editor';

export async function collectWorkflowProblems(
  document: WorkflowDocument
): Promise<Array<{ nodeId: string; title: string }>> {
  const nodes = document.getAllNodes();
  await Promise.all(nodes.map(async (n) => n.form?.validate()));
  const problemsMap = new Map<string, string>();
  for (const n of nodes.filter((n) => n.form?.state.invalid)) {
    const json: any = document.toNodeJSON(n);
    const title = json?.data?.title;
    problemsMap.set(n.id, title ? String(title) : n.id);
  }
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
    const requiredKeys: string[] = Array.isArray(inputs?.required) ? inputs.required : [];
    for (const k of requiredKeys) {
      const schema = inputs?.properties?.[k];
      const v = getVal(values?.[k]);
      if (isEmpty(schema, v)) {
        const title = json?.data?.title;
        problemsMap.set(String(json?.id), title ? String(title) : String(json?.id));
      }
    }
    const blocks = Array.isArray(json?.blocks) ? json.blocks : [];
    for (const b of blocks) validateNode(b);
  };
  for (const j of toJSONList) validateNode(j);
  return Array.from(problemsMap.entries()).map(([nodeId, title]) => ({ nodeId, title }));
}
