import { describe, expect, it } from 'vitest';

import { renderOverviewChatMarkdown } from '../overview-chat-markdown';

describe('renderOverviewChatMarkdown', () => {
  it('sanitizes dangerous html and javascript urls', () => {
    const html = renderOverviewChatMarkdown(
      'hello <script>alert(1)</script><img src="https://example.com/a.png" onerror="alert(1)" /><a href="javascript:alert(1)">bad</a>'
    );

    expect(html).toContain('hello');
    expect(html).not.toContain('<script');
    expect(html).not.toContain('onerror=');
    expect(html).not.toContain('javascript:alert(1)');
  });

  it('keeps code copy markup after sanitization', () => {
    const html = renderOverviewChatMarkdown('```ts\nconst x = 1;\n```');

    expect(html).toContain('overview-chat-code-copy');
    expect(html).toContain('data-copy-code=');
    expect(html).toContain('language-ts');
  });
});
