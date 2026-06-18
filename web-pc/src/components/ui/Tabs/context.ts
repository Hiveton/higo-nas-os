import type { InjectionKey, Ref } from 'vue';

/** Provided by Tabs.vue, consumed by TabPanel.vue to toggle visibility. */
export const TABS_ACTIVE_KEY: InjectionKey<Ref<string>> = Symbol('ui-tabs-active');
