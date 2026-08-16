import DOMPurify from 'dompurify';
import { marked } from 'marked';

export function sanitizeHtml(html) {
  return DOMPurify.sanitize(html || '', {
    USE_PROFILES: { html: true },
  });
}

export function renderSafeMarkdown(markdown) {
  return sanitizeHtml(marked.parse(markdown || ''));
}
