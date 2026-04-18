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

/** GET /api/v1/logs/runs/msgId（含 HTTP 状态，便于区分 404 / 写入延迟） */
export async function fetchRunLogsDetailed(msgId: string): Promise<{
  ok: boolean;
  status: number;
  data?: unknown;
}> {
  const url = `/logs/runs/msgId?msgId=${encodeURIComponent(msgId)}`;
  const resp = await requestRaw(url, { method: 'GET' });
  return { ok: resp.ok, status: resp.status, data: resp.data };
}

/** 兼容旧调用：失败时返回 undefined（勿与「空对象」混淆） */
export async function fetchRunLogs(msgId: string): Promise<any> {
  const r = await fetchRunLogsDetailed(msgId);
  return r.ok ? r.data : undefined;
}
