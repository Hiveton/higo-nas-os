import MarkdownIt from 'markdown-it';

// A single shared renderer. html:false is our sanitization — raw HTML in model
// output is escaped rather than rendered, so there is no XSS surface and we don't
// need a separate sanitizer. linkify turns bare URLs into links; breaks maps soft
// newlines to <br> for chat-friendly text.
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
});

// Open links in a new tab safely.
const defaultLinkOpen =
  md.renderer.rules.link_open ??
  ((tokens, idx, options, _env, self) => self.renderToken(tokens, idx, options));
md.renderer.rules.link_open = (tokens, idx, options, env, self) => {
  const token = tokens[idx];
  token.attrSet('target', '_blank');
  token.attrSet('rel', 'noopener noreferrer');
  return defaultLinkOpen(tokens, idx, options, env, self);
};

export function renderMarkdown(source: string): string {
  return md.render(source ?? '');
}

export function useMarkdown() {
  return { renderMarkdown };
}
