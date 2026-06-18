<script setup lang="ts">
import { UiSegmented } from '../../ui';

type InterfaceKey = 'uiTheme' | 'uiLocale' | 'windowRadius' | 'dockPosition' | 'dockStyle' | 'dockIconSize';
type Option = { value: string; label: string };

defineProps<{
  settings: Record<InterfaceKey, string>;
  themeOptions: Option[];
  localeOptions: Option[];
  radiusOptions: Option[];
  dockPositionOptions: Option[];
  dockStyleOptions: Option[];
  dockIconSizeOptions: Option[];
}>();

const emit = defineEmits<{
  (e: 'set-option', key: InterfaceKey, value: string): void;
}>();

function onSet(key: InterfaceKey, value: string | number) {
  emit('set-option', key, String(value));
}
</script>

<template>
  <div class="system-settings__panel system-settings__panel--interface">
    <section class="system-settings__account-card" aria-label="主题">
      <h4>主题</h4>
      <UiSegmented
        :model-value="settings.uiTheme"
        :options="themeOptions"
        @change="(v) => onSet('uiTheme', v)"
      />
    </section>

    <section class="system-settings__account-card" aria-label="语言 / Language">
      <h4>语言 / Language</h4>
      <UiSegmented
        :model-value="settings.uiLocale"
        :options="localeOptions"
        @change="(v) => onSet('uiLocale', v)"
      />
    </section>

    <section class="system-settings__account-card" aria-label="窗口圆角">
      <h4>窗口圆角</h4>
      <UiSegmented
        :model-value="settings.windowRadius"
        :options="radiusOptions"
        @change="(v) => onSet('windowRadius', v)"
      />
    </section>

    <section class="system-settings__account-card" aria-label="Dock 位置">
      <h4>Dock 位置</h4>
      <UiSegmented
        :model-value="settings.dockPosition"
        :options="dockPositionOptions"
        @change="(v) => onSet('dockPosition', v)"
      />
    </section>

    <section class="system-settings__account-card" aria-label="Dock 样式">
      <h4>Dock 样式</h4>
      <UiSegmented
        :model-value="settings.dockStyle"
        :options="dockStyleOptions"
        @change="(v) => onSet('dockStyle', v)"
      />
    </section>

    <section class="system-settings__account-card" aria-label="Dock 图标大小">
      <h4>Dock 图标大小</h4>
      <UiSegmented
        :model-value="settings.dockIconSize"
        :options="dockIconSizeOptions"
        @change="(v) => onSet('dockIconSize', v)"
      />
    </section>
  </div>
</template>
