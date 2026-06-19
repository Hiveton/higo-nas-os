<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Laptop, ShieldCheck, UserCog } from 'lucide-vue-next';
import { UiBadge, UiButton, UiInput, UiTabs, useToast } from '../../ui';
import type { TabItem } from '../../ui';
import { authStore } from '../../../stores/auth';
import { apiClient } from '../../../api/client';
import { ApiError } from '../../../api/runtime';
import type { AuthSession, MfaSetup } from '../../../api/types';

const toast = useToast();
const activeTab = ref('profile');
const tabs: TabItem[] = [
  { key: 'profile', label: '资料', icon: UserCog },
  { key: 'security', label: '安全', icon: ShieldCheck },
  { key: 'sessions', label: '会话与设备', icon: Laptop },
];

const user = computed(() => authStore.currentUser.value);
const roleLabel = computed(() => roleText(user.value?.role));
const quotaLabel = computed(() => {
  const bytes = user.value?.quotaBytes ?? 0;
  return bytes ? `${(bytes / 1024 / 1024 / 1024).toFixed(0)} GB` : '不限';
});

function roleText(role?: string) {
  return role === 'admin' ? '管理员' : role === 'user' ? '成员' : role === 'guest' ? '访客' : role ?? '—';
}

// change password
const currentPassword = ref('');
const newPassword = ref('');
const confirmPassword = ref('');
const changing = ref(false);

async function submitPassword() {
  if (!currentPassword.value || !newPassword.value) {
    toast.warning('请填写当前密码与新密码');
    return;
  }
  if (newPassword.value !== confirmPassword.value) {
    toast.error('两次输入的新密码不一致');
    return;
  }
  changing.value = true;
  try {
    await authStore.changePassword({ currentPassword: currentPassword.value, newPassword: newPassword.value });
    toast.success('密码已更新');
    currentPassword.value = newPassword.value = confirmPassword.value = '';
  } catch (reason) {
    toast.error(reason instanceof ApiError ? reason.message : '修改失败，请重试');
  } finally {
    changing.value = false;
  }
}

// MFA
const mfaEnabled = computed(() => user.value?.mfaEnabled ?? false);
const mfaSetup = ref<MfaSetup | null>(null);
const mfaCode = ref('');
const mfaDisablePassword = ref('');
const mfaBusy = ref(false);

async function startMfaSetup() {
  mfaBusy.value = true;
  try {
    mfaSetup.value = await apiClient.auth.mfaSetup();
  } catch {
    toast.error('无法开始两步验证设置');
  } finally {
    mfaBusy.value = false;
  }
}

async function confirmMfa() {
  if (mfaCode.value.trim().length !== 6) {
    toast.warning('请输入 6 位动态码');
    return;
  }
  mfaBusy.value = true;
  try {
    await apiClient.auth.mfaEnable(mfaCode.value.trim());
    await authStore.refresh();
    mfaSetup.value = null;
    mfaCode.value = '';
    toast.success('两步验证已开启');
  } catch {
    toast.error('动态码不正确，请重试');
  } finally {
    mfaBusy.value = false;
  }
}

async function disableMfa() {
  if (!mfaDisablePassword.value) {
    toast.warning('请输入当前密码');
    return;
  }
  mfaBusy.value = true;
  try {
    await apiClient.auth.mfaDisable(mfaDisablePassword.value);
    await authStore.refresh();
    mfaDisablePassword.value = '';
    toast.success('两步验证已关闭');
  } catch {
    toast.error('关闭失败，请检查密码');
  } finally {
    mfaBusy.value = false;
  }
}

// sessions
const sessions = ref<AuthSession[]>([]);
const loadingSessions = ref(false);

async function loadSessions() {
  loadingSessions.value = true;
  try {
    sessions.value = await apiClient.auth.listSessions();
  } catch {
    sessions.value = [];
  } finally {
    loadingSessions.value = false;
  }
}

async function revoke(session: AuthSession) {
  if (session.current) return;
  try {
    await apiClient.auth.revokeSession(session.id);
    toast.success('已撤销该会话');
    await loadSessions();
  } catch {
    toast.error('撤销失败，请重试');
  }
}

onMounted(loadSessions);
</script>

<template>
  <div class="uc-panel">
    <header class="uc-id">
      <div class="uc-id__avatar">{{ (user?.displayName || user?.username || 'H').charAt(0).toUpperCase() }}</div>
      <div>
        <h3 class="uc-id__name">{{ user?.displayName || user?.username || '当前用户' }}</h3>
        <UiBadge tone="info">{{ roleLabel }}</UiBadge>
      </div>
    </header>

    <UiTabs v-model="activeTab" :tabs="tabs" />

    <div v-show="activeTab === 'profile'">
      <dl class="uc-grid">
        <div><dt>用户名</dt><dd>{{ user?.username }}</dd></div>
        <div><dt>显示名</dt><dd>{{ user?.displayName }}</dd></div>
        <div><dt>角色</dt><dd>{{ roleLabel }}</dd></div>
        <div><dt>状态</dt><dd>{{ user?.status === 'active' ? '正常' : user?.status }}</dd></div>
        <div><dt>空间配额</dt><dd>{{ quotaLabel }}</dd></div>
        <div><dt>所属组</dt><dd>{{ (user?.groups ?? []).join('、') || '—' }}</dd></div>
      </dl>
    </div>

    <div v-show="activeTab === 'security'">
      <form class="uc-form" @submit.prevent="submitPassword">
        <label>当前密码</label>
        <UiInput v-model="currentPassword" type="password" placeholder="请输入当前密码" />
        <label>新密码</label>
        <UiInput v-model="newPassword" type="password" placeholder="至少 8 位，含字母与数字" />
        <label>确认新密码</label>
        <UiInput v-model="confirmPassword" type="password" placeholder="再次输入新密码" />
        <div class="uc-actions"><UiButton type="submit" :loading="changing">更新密码</UiButton></div>
      </form>

      <div class="uc-mfa">
        <div class="uc-mfa__head">
          <strong>两步验证（TOTP）</strong>
          <UiBadge :tone="mfaEnabled ? 'success' : 'neutral'">{{ mfaEnabled ? '已开启' : '未开启' }}</UiBadge>
        </div>
        <template v-if="!mfaEnabled">
          <UiButton v-if="!mfaSetup" variant="soft" :loading="mfaBusy" @click="startMfaSetup">开启两步验证</UiButton>
          <div v-else class="uc-form">
            <p class="uc-muted">用身份验证器 App 录入密钥，再输入 6 位动态码确认：</p>
            <code class="uc-secret">{{ mfaSetup.secret }}</code>
            <UiInput v-model="mfaCode" placeholder="6 位动态码" inputmode="numeric" />
            <div class="uc-actions">
              <UiButton :loading="mfaBusy" @click="confirmMfa">确认开启</UiButton>
              <UiButton variant="ghost" @click="mfaSetup = null">取消</UiButton>
            </div>
          </div>
        </template>
        <div v-else class="uc-form">
          <label>输入当前密码以关闭两步验证</label>
          <UiInput v-model="mfaDisablePassword" type="password" placeholder="当前密码" />
          <div class="uc-actions">
            <UiButton variant="soft" tone="danger" :loading="mfaBusy" @click="disableMfa">关闭两步验证</UiButton>
          </div>
        </div>
      </div>
    </div>

    <div v-show="activeTab === 'sessions'">
      <p v-if="loadingSessions" class="uc-muted">加载中…</p>
      <p v-else-if="sessions.length === 0" class="uc-muted">暂无活动会话。</p>
      <ul v-else class="uc-list">
        <li v-for="session in sessions" :key="session.id" class="uc-list__row">
          <div>
            <strong>{{ session.device || '未知设备' }}</strong>
            <UiBadge v-if="session.current" tone="success">当前</UiBadge>
            <div class="uc-list__meta">{{ session.ipAddress || '—' }} · 最近活跃 {{ session.lastSeenAt }}</div>
          </div>
          <UiButton variant="ghost" tone="danger" size="sm" :disabled="session.current" @click="revoke(session)">撤销</UiButton>
        </li>
      </ul>
    </div>
  </div>
</template>
