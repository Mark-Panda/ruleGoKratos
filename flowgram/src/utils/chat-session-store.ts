export type SessionStoreSession = {
  id: string;
  title: string;
  updatedAt: number;
  messages: Array<{ role: string; content: string }>;
  historyStartIndex: number;
};

export type SessionStore<T extends SessionStoreSession = SessionStoreSession> = {
  sessions: T[];
  activeId: string | null;
};

export function patchSessionById<T extends SessionStoreSession>(
  prev: SessionStore<T>,
  sessionId: string | null | undefined,
  recipe: (session: T) => T
): SessionStore<T> {
  if (!sessionId) return prev;
  return {
    ...prev,
    sessions: prev.sessions.map((session) => (session.id === sessionId ? recipe(session) : session)),
  };
}
