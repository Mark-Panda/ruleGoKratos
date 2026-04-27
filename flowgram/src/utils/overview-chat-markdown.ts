import hljs from 'highlight.js/lib/common';
import { Marked } from 'marked';
import sanitizeHtml from 'sanitize-html';
import type { IOptions } from 'sanitize-html';

function escapeHtml(raw: string): string {
  return raw
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

const renderer = {
  code(code: string, infostring?: string) {
    const rawLang = (infostring || '').trim().split(/\s+/)[0] || '';
    let lang = rawLang.toLowerCase();
    let highlighted = escapeHtml(code || '');
    try {
      if (lang && hljs.getLanguage(lang)) {
        highlighted = hljs.highlight(code || '', { language: lang, ignoreIllegals: true }).value;
      } else {
        const auto = hljs.highlightAuto(code || '');
        highlighted = auto.value || highlighted;
        if (!lang && auto.language) lang = auto.language;
      }
    } catch {
      highlighted = escapeHtml(code || '');
    }
    if (!lang) lang = 'text';
    const encoded = encodeURIComponent(code || '');
    return `
<div class="overview-chat-code-wrap">
  <div class="overview-chat-code-toolbar">
    <span class="overview-chat-code-lang">${escapeHtml(lang)}</span>
    <button class="overview-chat-code-copy" type="button" data-copy-code="${encoded}">复制代码</button>
  </div>
  <pre><code class="hljs language-${escapeHtml(lang)}">${highlighted}</code></pre>
</div>`;
  },
};

const marked = new Marked({
  breaks: true,
  renderer,
});

const allowedTags = Array.from(
  new Set([
    ...sanitizeHtml.defaults.allowedTags,
    'img',
    'div',
    'span',
    'button',
    'table',
    'thead',
    'tbody',
    'tr',
    'th',
    'td',
    'pre',
    'code',
  ])
);

const allowedAttributes: IOptions['allowedAttributes'] = {
  ...sanitizeHtml.defaults.allowedAttributes,
  a: ['href', 'name', 'target', 'rel'],
  img: ['src', 'alt', 'title'],
  div: ['class'],
  span: ['class'],
  code: ['class'],
  pre: ['class'],
  button: ['class', 'type', 'data-copy-code'],
  th: ['class'],
  td: ['class'],
};

export function renderOverviewChatMarkdown(raw: string): string {
  try {
    const parsed = String(marked.parse(raw || ''));
    return sanitizeHtml(parsed, {
      allowedTags,
      allowedAttributes,
      allowedSchemes: ['http', 'https', 'mailto'],
      allowedSchemesAppliedToAttributes: ['href', 'src'],
    });
  } catch {
    return `<p>${escapeHtml(raw || '')}</p>`;
  }
}
