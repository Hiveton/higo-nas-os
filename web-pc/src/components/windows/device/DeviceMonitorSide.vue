<script setup lang="ts">
import {
  AlertTriangle,
  Bell,
  CheckCircle2,
  Plus,
  RefreshCw,
  ServerCog,
  Volume2,
  VolumeX,
} from 'lucide-vue-next';
import type { Component } from 'vue';
import type { Alert, ServiceStatus, SystemLog } from '../../../api/types';
import { UiBadge, UiButton } from '../../ui';
import type { UiTone } from '../../ui';

type ViewService = ServiceStatus & { icon: Component; tone: string };

function severityTone(severity?: string): UiTone {
  const text = String(severity ?? '');
  if (text.includes('高')) return 'danger';
  if (text.includes('中')) return 'warning';
  return 'info';
}

defineProps<{
  diagnosticState: string;
  selectedLog: SystemLog;
  alerts: readonly Alert[];
  selectedAlertId: string | undefined;
  selectedAlert: Alert;
  serviceStates: readonly ViewService[];
}>();

const emit = defineEmits<{
  (e: 'refresh-diagnostics'): void;
  (e: 'create-alert'): void;
  (e: 'select-alert', id: string): void;
  (e: 'mute-alert'): void;
}>();
</script>

<template>
  <aside class="device-monitor__side" aria-label="告警和诊断">
    <section class="device-monitor__diagnostic">
      <header>
        <h3><CheckCircle2 :size="15" /> 诊断</h3>
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="emit('refresh-diagnostics')">重跑</UiButton>
      </header>
      <p>{{ diagnosticState }}</p>
      <div>
        <span>选中日志</span>
        <strong>{{ selectedLog.source }}</strong>
        <small>{{ selectedLog.message }}</small>
      </div>
    </section>

    <section class="device-monitor__alerts">
      <header>
        <h3><Bell :size="15" /> 告警</h3>
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="Plus" @click="emit('create-alert')">创建</UiButton>
      </header>
      <button
        v-for="alert in alerts"
        :key="alert.id ?? alert.title"
        class="device-monitor__alert"
        :class="{ 'device-monitor__alert--active': alert.id === selectedAlertId, 'device-monitor__alert--muted': alert.muted }"
        type="button"
        @click="alert.id && emit('select-alert', alert.id)"
      >
        <AlertTriangle :size="15" />
        <span>
          <strong>{{ alert.title }}</strong>
          <small>
            <UiBadge :tone="severityTone(alert.severity)" variant="soft" size="sm">{{ alert.severity }}</UiBadge>
            {{ alert.state }}
          </small>
        </span>
      </button>

      <div class="device-monitor__alert-detail">
        <strong>{{ selectedAlert.title }}</strong>
        <p>{{ selectedAlert.detail }}</p>
        <UiButton
          variant="soft"
          tone="neutral"
          size="sm"
          :icon-left="selectedAlert.muted ? Volume2 : VolumeX"
          @click="emit('mute-alert')"
        >
          {{ selectedAlert.muted ? '恢复提醒' : '静音告警' }}
        </UiButton>
      </div>
    </section>

    <section class="device-monitor__services" aria-label="容器、应用、任务、备份和下载状态">
      <header>
        <h3><ServerCog :size="15" /> 服务状态</h3>
      </header>
      <article v-for="service in serviceStates" :key="service.label" :class="`device-monitor__service--${service.tone}`">
        <component :is="service.icon" :size="16" />
        <div>
          <strong>{{ service.label }} · {{ service.value }}</strong>
          <span>{{ service.detail }}</span>
        </div>
      </article>
    </section>
  </aside>
</template>
