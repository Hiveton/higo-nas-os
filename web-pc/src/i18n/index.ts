import { createI18n } from 'vue-i18n';
import zhCN from './locales/zh-CN';
import enUS from './locales/en-US';

export type AppLocale = 'zh-CN' | 'en-US';

export const SUPPORTED_LOCALES: { value: AppLocale; label: string }[] = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
];

export const i18n = createI18n({
  legacy: false,
  locale: 'zh-CN',
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
  // The app is mid-migration: many strings are still inline literals.
  // Suppress noisy warnings until extraction is complete.
  missingWarn: false,
  fallbackWarn: false,
});

/** Switch the active locale (call from settings). */
export function setLocale(locale: AppLocale) {
  i18n.global.locale.value = locale;
  document.documentElement.setAttribute('lang', locale);
}
