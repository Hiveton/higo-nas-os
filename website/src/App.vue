<script setup lang="ts">
import {
  AppWindow,
  ArchiveRestore,
  Bot,
  Boxes,
  ChevronRight,
  Cloud,
  Database,
  Download,
  FolderClosed,
  HardDrive,
  Network,
  Play,
  ShieldCheck,
  Sparkles,
} from 'lucide-vue-next';
import { featureCards } from './data/features';
import { aiPillars, metrics, navItems, productUrl, securitySteps } from './data/sections';

const productIcons = [FolderClosed, HardDrive, Sparkles, ArchiveRestore, Download, Boxes, Cloud, Network];
const heroTags = ['正版可部署', '硬件兼容', 'AI 文件管家', 'Docker 应用', '安全回滚'];
const highlights = [
  { title: '本地优先', detail: '文件、照片、备份和权限留在自己的 NAS。', icon: Database },
  { title: 'AI 原生', detail: '语义搜索、自动整理和 Agent 工作台围绕真实数据运行。', icon: Bot },
  { title: '开放生态', detail: '共享协议、应用中心、Docker 与远程访问组合成 NASOS。', icon: AppWindow },
];
const ecosystemItems = [
  { title: '应用中心', detail: '常用服务开箱即用，安装、更新、权限和资源统一管理。', icon: AppWindow },
  { title: 'Docker / Compose', detail: '容器、端口、卷挂载和日志面向进阶用户开放。', icon: Boxes },
  { title: '共享协议', detail: 'SMB、NFS、WebDAV、DLNA 覆盖家庭与团队访问。', icon: Network },
  { title: '远程访问', detail: '设备绑定、直连/中继、域名令牌和登录告警。', icon: Cloud },
];

function scrollToSection(id: string) {
  document.querySelector(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

function openProduct() {
  window.open(productUrl, '_blank', 'noopener,noreferrer');
}
</script>

<template>
  <main class="site-shell">
    <header class="site-header" aria-label="HiGoOS 官网导航">
      <a class="brand" href="#top" aria-label="HiGoOS 首页">
        <span class="brand__mark">H</span>
        <strong>HiGoOS</strong>
      </a>
      <nav class="site-nav">
        <a v-for="item in navItems" :key="item.href" :href="item.href">{{ item.label }}</a>
      </nav>
      <div class="header-actions">
        <button class="nav-button" type="button" @click="openProduct">体验桌面</button>
        <button class="nav-button nav-button--primary" type="button" @click="scrollToSection('#product')">查看产品</button>
      </div>
    </header>

    <section id="top" class="hero">
      <p class="notice">AI 原生家庭 NAS 系统，面向真实部署</p>
      <h1>HiGoOS</h1>
      <div class="hero-tags" aria-label="HiGoOS 核心卖点">
        <span v-for="tag in heroTags" :key="tag">{{ tag }}</span>
      </div>
      <div class="hero-actions">
        <button class="hero-button hero-button--primary" type="button" @click="scrollToSection('#product')">
          查看产品
          <ChevronRight :size="20" />
        </button>
        <button class="hero-button" type="button" @click="openProduct">
          体验桌面
          <Play :size="20" />
        </button>
      </div>
    </section>

    <section class="metric-row" aria-label="HiGoOS 状态摘要">
      <article v-for="metric in metrics" :key="metric.label">
        <span>{{ metric.label }}</span>
        <strong>{{ metric.value }}</strong>
      </article>
    </section>

    <section id="product" class="section">
      <div class="section-title">
        <p>Product</p>
        <h2>存储、媒体、备份和应用生态，一套系统完成</h2>
      </div>
      <div class="feature-grid">
        <article v-for="(feature, index) in featureCards" :key="feature.title" class="feature-card">
          <component :is="productIcons[index]" :size="26" />
          <h3>{{ feature.title }}</h3>
          <p>{{ feature.detail }}</p>
        </article>
      </div>
    </section>

    <section id="ai" class="section split-section">
      <div>
        <p class="section-kicker">AI Native</p>
        <h2>AI 能看见文件，但只能在权限内行动</h2>
        <p class="section-copy">
          HiGoOS 把语义搜索、智能整理、AI 助手和 Agent 工作台做成系统能力。建议先预览，执行要确认，结果能审计。
        </p>
        <div class="pill-list">
          <span v-for="item in aiPillars" :key="item">{{ item }}</span>
        </div>
      </div>
      <div class="mini-grid">
        <article v-for="item in highlights" :key="item.title">
          <component :is="item.icon" :size="28" />
          <h3>{{ item.title }}</h3>
          <p>{{ item.detail }}</p>
        </article>
      </div>
    </section>

    <section id="security" class="section security-section">
      <div class="section-title">
        <p>Governance</p>
        <h2>安全确认与回滚，不让 AI 操作失控</h2>
      </div>
      <div class="security-steps">
        <span v-for="step in securitySteps" :key="step">{{ step }}</span>
      </div>
    </section>

    <section id="ecosystem" class="section">
      <div class="section-title">
        <p>Ecosystem</p>
        <h2>面向家庭与小团队的开放 NASOS</h2>
      </div>
      <div class="ecosystem-grid">
        <article v-for="item in ecosystemItems" :key="item.title">
          <component :is="item.icon" :size="28" />
          <h3>{{ item.title }}</h3>
          <p>{{ item.detail }}</p>
        </article>
      </div>
    </section>

    <section id="deployment" class="final-cta">
      <ShieldCheck :size="34" />
      <h2>从局域网发现，到 Web 初始化，再进入 HiGoOS 桌面</h2>
      <p>桌面扫描工具负责发现和配置设备，真正的 NAS 业务在 HiGoOS Web 桌面中完成。</p>
      <div class="hero-actions">
        <button class="hero-button hero-button--primary" type="button" @click="openProduct">体验桌面</button>
        <button class="hero-button" type="button" @click="scrollToSection('#product')">了解功能</button>
      </div>
    </section>
  </main>
</template>
