import { beforeEach, describe, expect, it } from 'vitest';

import { loadStoredManagedAgentId, saveStoredManagedAgentId } from '../managed-agent-storage';

const storage = new Map<string, string>();

beforeEach(() => {
  storage.clear();
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: (key: string) => (storage.has(key) ? storage.get(key)! : null),
      setItem: (key: string, value: string) => {
        storage.set(key, value);
      },
      removeItem: (key: string) => {
        storage.delete(key);
      },
    },
    configurable: true,
  });
});

describe('managed-agent-storage', () => {
  it('loads persisted managed agent id', () => {
    storage.set('flowgram-overview-chat-managed-agent-v1', JSON.stringify(23));

    expect(loadStoredManagedAgentId()).toBe(23);
  });

  it('saves valid managed agent id and clears invalid value', () => {
    saveStoredManagedAgentId(42);
    expect(storage.get('flowgram-overview-chat-managed-agent-v1')).toBe(JSON.stringify(42));

    saveStoredManagedAgentId(0);
    expect(storage.has('flowgram-overview-chat-managed-agent-v1')).toBe(false);
  });
});
