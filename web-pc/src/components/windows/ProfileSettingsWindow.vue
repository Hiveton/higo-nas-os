<script setup lang="ts">
// Personal settings (DSM "个人设置" from the top-right user menu): self-service
// profile, password, MFA and own sessions. Available to every logged-in user —
// distinct from the admin-only 用户中心.
import MyAccountPanel from './user-center/MyAccountPanel.vue';
import './user-center/user-center.css';
</script>

<template>
  <div class="profile-settings">
    <MyAccountPanel />
  </div>
</template>

<style scoped>
.profile-settings {
  height: 100%;
  overflow: auto;
  padding: 18px 22px;
}
</style>
