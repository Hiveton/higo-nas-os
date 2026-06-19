<script setup lang="ts">
import { computed, ref } from 'vue';
import { Lock, ShieldCheck, User } from 'lucide-vue-next';
import { UiButton, UiCheckbox, UiInput } from './ui';
import { authStore } from '../stores/auth';
import { ApiError } from '../api/runtime';
import wallpaperUrl from '../assets/higoos-dock/wallpaper.png';

const emit = defineEmits<{ authenticated: [] }>();

const username = ref('');
const password = ref('');
const code = ref('');
const rememberDevice = ref(false);
const localError = ref('');
const mfaStep = ref(false);

const loading = computed(() => authStore.loading.value);
const backgroundStyle = computed(() => ({ backgroundImage: `url(${wallpaperUrl})` }));

async function submit() {
  localError.value = '';
  if (!username.value.trim() || !password.value) {
    localError.value = '请输入用户名和密码。';
    return;
  }
  if (mfaStep.value && !code.value.trim()) {
    localError.value = '请输入身份验证器中的 6 位动态码。';
    return;
  }
  try {
    await authStore.login({
      username: username.value.trim(),
      password: password.value,
      code: mfaStep.value ? code.value.trim() : undefined,
      rememberDevice: rememberDevice.value,
    });
    emit('authenticated');
  } catch (reason) {
    if (reason instanceof ApiError) {
      if (reason.code === 'mfa_required') {
        mfaStep.value = true;
        localError.value = '请输入身份验证器中的 6 位动态码。';
        return;
      }
      if (reason.code === 'mfa_invalid') localError.value = '动态码不正确，请重试。';
      else if (reason.status === 401) localError.value = '用户名或密码错误。';
      else if (reason.status === 403) localError.value = '账号已被停用或锁定，请联系管理员。';
      else localError.value = `登录失败：${reason.message}`;
    } else {
      localError.value = '无法连接到 HiGoOS 服务，请稍后重试。';
    }
  }
}
</script>

<template>
  <div class="login" :style="backgroundStyle">
    <div class="login__scrim" />
    <section class="login__card" aria-label="HiGoOS 登录">
      <div class="login__brand">
        <ShieldCheck :size="34" class="login__logo" />
        <div>
          <h1 class="login__title">HiGoOS</h1>
          <p class="login__subtitle">AI 原生 NAS · 用户中心</p>
        </div>
      </div>

      <form class="login__form" @submit.prevent="submit">
        <label class="login__label" for="login-username">用户名</label>
        <UiInput
          id="login-username"
          v-model="username"
          :prefix-icon="User"
          placeholder="请输入用户名"
          autocomplete="username"
          @enter="submit"
        />

        <label class="login__label" for="login-password">密码</label>
        <UiInput
          id="login-password"
          v-model="password"
          type="password"
          :prefix-icon="Lock"
          placeholder="请输入密码"
          autocomplete="current-password"
          @enter="submit"
        />

        <template v-if="mfaStep">
          <label class="login__label" for="login-code">动态验证码</label>
          <UiInput
            id="login-code"
            v-model="code"
            :prefix-icon="ShieldCheck"
            placeholder="身份验证器 6 位动态码"
            inputmode="numeric"
            autocomplete="one-time-code"
            @enter="submit"
          />
        </template>

        <div class="login__row">
          <UiCheckbox v-model="rememberDevice">信任此设备</UiCheckbox>
        </div>

        <p v-if="localError" class="login__error" role="alert">{{ localError }}</p>

        <UiButton type="submit" size="lg" block :loading="loading">登录</UiButton>
      </form>

      <p class="login__hint">首次登录请使用管理员账号；初始密码见服务器启动日志。</p>
    </section>
  </div>
</template>

<style scoped>
.login {
  position: fixed;
  inset: 0;
  display: grid;
  place-items: center;
  background-size: cover;
  background-position: center;
  z-index: 1000;
}

.login__scrim {
  position: absolute;
  inset: 0;
  background: radial-gradient(120% 120% at 50% 0%, rgba(15, 23, 42, 0.35), rgba(2, 6, 23, 0.72));
  backdrop-filter: blur(6px);
}

.login__card {
  position: relative;
  width: min(380px, calc(100vw - 48px));
  padding: 32px 30px 26px;
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.14);
  border: 1px solid rgba(255, 255, 255, 0.22);
  box-shadow: 0 24px 60px rgba(2, 6, 23, 0.5);
  backdrop-filter: blur(26px) saturate(160%);
  color: #f8fafc;
}

.login__brand {
  display: flex;
  align-items: center;
  gap: 14px;
  margin-bottom: 24px;
}

.login__logo {
  color: #7dd3fc;
}

.login__title {
  margin: 0;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 0.5px;
}

.login__subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: rgba(248, 250, 252, 0.7);
}

.login__form {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.login__label {
  font-size: 12px;
  font-weight: 600;
  color: rgba(248, 250, 252, 0.82);
  margin-top: 6px;
}

.login__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 8px 0 4px;
  font-size: 13px;
}

.login__error {
  margin: 2px 0 6px;
  font-size: 12.5px;
  color: #fca5a5;
}

.login__hint {
  margin: 18px 0 0;
  font-size: 11.5px;
  line-height: 1.5;
  color: rgba(248, 250, 252, 0.58);
  text-align: center;
}
</style>
