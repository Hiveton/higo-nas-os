/**
 * Shared variant unions for the in-house UI library.
 * These map to the CSS custom properties defined in src/styles/tokens.css.
 */

export type UiTone =
  | 'neutral'
  | 'primary'
  | 'success'
  | 'warning'
  | 'danger'
  | 'info';

export type UiSize = 'sm' | 'md' | 'lg';
