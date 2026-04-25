export const STORAGE_MANAGED_AGENT_KEY = 'flowgram-overview-chat-managed-agent-v1';

function getStorage(): Storage | undefined {
  const candidate = globalThis.localStorage;
  return candidate ?? (typeof window !== 'undefined' ? window.localStorage : undefined);
}

export function loadStoredManagedAgentId(): number {
  try {
    const raw = getStorage()?.getItem(STORAGE_MANAGED_AGENT_KEY) ?? null;
    if (!raw) return 0;
    const id = Number(JSON.parse(raw));
    return Number.isFinite(id) && id > 0 ? id : 0;
  } catch {
    return 0;
  }
}

export function saveStoredManagedAgentId(id: number) {
  try {
    const storage = getStorage();
    if (!storage) return;
    if (id > 0) {
      storage.setItem(STORAGE_MANAGED_AGENT_KEY, JSON.stringify(id));
      return;
    }
    storage.removeItem(STORAGE_MANAGED_AGENT_KEY);
  } catch {
    /* ignore */
  }
}
