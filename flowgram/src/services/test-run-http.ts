import { requestRaw } from './http';

export async function executeTestRun(params: {
  ruleId: string;
  msgType: string;
  metadata?: string;
  headers?: Record<string, any>;
  body?: any;
  debugMode?: boolean;
  msgId: string;
}): Promise<{ ok: boolean; status: number; data: any }> {
  const { ruleId, msgType, metadata, headers, body, debugMode, msgId } = params;
  const qs = [
    `debugMode=${debugMode ? 'true' : 'false'}`,
    metadata ? metadata : '',
    `msgId=${encodeURIComponent(msgId)}`,
  ]
    .filter(Boolean)
    .join('&');
  const url = `/rules/${encodeURIComponent(ruleId)}/notify/${encodeURIComponent(msgType)}${
    qs ? `?${qs}` : ''
  }`;
  const hdrs: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  Object.keys(headers || {}).forEach((k) => {
    const v = (headers as any)[k];
    if (v !== undefined && v !== null) hdrs[k] = String(v);
  });
  return requestRaw(url, { method: 'POST', headers: hdrs, body });
}

export async function fetchRunLogs(msgId: string): Promise<any> {
  const url = `/logs/runs/msgId?msgId=${encodeURIComponent(msgId)}`;
  const resp = await requestRaw(url, { method: 'GET' });
  if (resp.ok) return resp.data;
  return {};
}
