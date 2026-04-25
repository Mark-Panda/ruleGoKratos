import { describe, expect, it } from 'vitest';

import { patchSessionById } from '../chat-session-store';

describe('patchSessionById', () => {
  it('updates the targeted session instead of current active session', () => {
    const store = {
      activeId: 'session-b',
      sessions: [
        {
          id: 'session-a',
          title: 'A',
          updatedAt: 1,
          messages: [{ role: 'assistant', content: '' }],
          historyStartIndex: 0,
        },
        {
          id: 'session-b',
          title: 'B',
          updatedAt: 1,
          messages: [{ role: 'assistant', content: 'keep' }],
          historyStartIndex: 0,
        },
      ],
    };

    const next = patchSessionById(store, 'session-a', (session) => ({
      ...session,
      messages: [{ role: 'assistant', content: 'patched' }],
    }));

    expect(next.sessions[0].messages[0]?.content).toBe('patched');
    expect(next.sessions[1].messages[0]?.content).toBe('keep');
  });
});
