<template>
  <div class="app-layout">
    <div class="sidebar" :class="{ collapsed }">
      <div class="sidebar-logo">
        <span v-if="!collapsed" class="logo-text">VirtPanel</span>
        <span v-else class="logo-text">V</span>
      </div>
      <nav class="sidebar-nav">
        <div
          v-for="item in menuItems"
          :key="item.key"
          class="nav-item"
          :class="{ active: currentRoute === item.key }"
          @click="onMenuClick(item.key)"
        >
          <component :is="item.icon" class="nav-icon" />
          <span v-if="!collapsed" class="nav-label">{{ item.label }}</span>
        </div>
      </nav>
      <div class="sidebar-bottom" @click="collapsed = !collapsed">
        <icon-left v-if="!collapsed" class="nav-icon" />
        <icon-right v-else class="nav-icon" />
      </div>
    </div>
    <div class="main-area">
      <header class="app-header">
        <h1 class="page-title">{{ pageTitle }}</h1>
      </header>
      <main class="app-content">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  IconDashboard, IconDesktop, IconWifi, IconStorage, IconHistory, IconFile,
  IconLeft, IconRight,
} from '@arco-design/web-vue/es/icon'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)

const menuItems = [
  { key: 'dashboard', label: '概览', icon: IconDashboard },
  { key: 'vm', label: '虚拟机', icon: IconDesktop },
  { key: 'snapshot', label: '快照', icon: IconHistory },
  { key: 'network', label: '网络', icon: IconWifi },
  { key: 'storage', label: '存储', icon: IconStorage },
  { key: 'iso', label: '镜像', icon: IconFile },
]

const currentRoute = computed(() => {
  const name = route.name as string
  return name === 'vm-detail' ? 'vm' : name
})

const titles: Record<string, string> = {
  dashboard: '概览', vm: '虚拟机', 'vm-detail': '虚拟机详情',
  snapshot: '快照', network: '网络', storage: '存储', iso: '镜像',
}
const pageTitle = computed(() => titles[route.name as string] || '')
const onMenuClick = (key: string) => router.push({ name: key })
</script>

<style scoped>
.app-layout {
  height: 100vh;
  display: flex;
  overflow: hidden;
}

.sidebar {
  width: 220px;
  background: #f0f0f5;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease;
  flex-shrink: 0;
  border-right: 1px solid rgba(0,0,0,0.06);
}
.sidebar.collapsed { width: 60px; }

.sidebar-logo {
  height: 54px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: #1c1c1e;
  letter-spacing: 0.5px;
}

.sidebar-nav {
  flex: 1;
  padding: 6px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.collapsed .sidebar-nav { padding: 6px; }

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.12s;
  color: #48484a;
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
}
.collapsed .nav-item { justify-content: center; padding: 9px; }

.nav-item:hover { background: rgba(0,0,0,0.04); }
.nav-item.active {
  background: #007AFF;
  color: #fff;
}
.nav-item.active .nav-icon { color: #fff; }

.nav-icon { font-size: 18px; flex-shrink: 0; color: #636366; }
.nav-label { font-weight: 500; }

.sidebar-bottom {
  padding: 12px;
  display: flex;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
}
.sidebar-bottom .nav-icon { color: #aeaeb2; font-size: 14px; }

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #f9f9fb;
}
.app-header {
  height: 54px;
  display: flex;
  align-items: center;
  padding: 0 28px;
  background: #fff;
  border-bottom: 1px solid rgba(0,0,0,0.06);
  flex-shrink: 0;
}
.page-title {
  font-size: 18px;
  font-weight: 700;
  color: #1c1c1e;
  margin: 0;
}
.app-content {
  flex: 1;
  padding: 24px 28px;
  overflow-y: auto;
}
</style>
