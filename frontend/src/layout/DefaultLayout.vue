<template>
  <a-layout class="app-layout">
    <div class="sidebar" :class="{ collapsed }">
      <div class="sidebar-logo" @click="collapsed = !collapsed">
        <div class="logo-dot" />
        <transition name="fade">
          <span v-if="!collapsed" class="logo-text">虚拟化</span>
        </transition>
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
          <transition name="fade">
            <span v-if="!collapsed" class="nav-label">{{ item.label }}</span>
          </transition>
        </div>
      </nav>
      <div class="sidebar-footer" @click="collapsed = !collapsed">
        <icon-left v-if="!collapsed" class="nav-icon" style="color:rgba(255,255,255,0.4)" />
        <icon-right v-else class="nav-icon" style="color:rgba(255,255,255,0.4)" />
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
  </a-layout>
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
  flex-direction: row;
  overflow: hidden;
}
.sidebar {
  width: 200px;
  background: linear-gradient(180deg, #1a3a5c 0%, #1e4976 100%);
  display: flex;
  flex-direction: column;
  transition: width 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  flex-shrink: 0;
}
.sidebar.collapsed { width: 56px; }
.sidebar-logo {
  height: 52px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  gap: 10px;
  cursor: pointer;
  flex-shrink: 0;
}
.collapsed .sidebar-logo { justify-content: center; padding: 0; }
.logo-dot {
  width: 10px; height: 10px;
  border-radius: 50%;
  background: #5ac8fa;
  flex-shrink: 0;
}
.logo-text {
  font-size: 14px;
  font-weight: 600;
  color: rgba(255,255,255,0.9);
  letter-spacing: 0.5px;
  white-space: nowrap;
}
.sidebar-nav {
  flex: 1;
  padding: 8px 6px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.collapsed .sidebar-nav { padding: 8px 4px; }
.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border-radius: 7px;
  cursor: pointer;
  transition: all 0.15s ease;
  color: rgba(255,255,255,0.5);
  white-space: nowrap;
  overflow: hidden;
  font-size: 13px;
}
.collapsed .nav-item { justify-content: center; padding: 8px; }
.nav-item:hover {
  background: rgba(255,255,255,0.08);
  color: rgba(255,255,255,0.8);
}
.nav-item.active {
  background: rgba(255,255,255,0.12);
  color: #fff;
}
.nav-icon { font-size: 17px; flex-shrink: 0; }
.nav-label { font-weight: 500; }
.sidebar-footer {
  padding: 12px;
  display: flex;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
}
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #f2f2f7;
}
.app-header {
  height: 52px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: rgba(255,255,255,0.8);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 0.5px solid rgba(0,0,0,0.08);
  flex-shrink: 0;
}
.page-title {
  font-size: 17px;
  font-weight: 600;
  color: #1c1c1e;
  margin: 0;
}
.app-content {
  flex: 1;
  padding: 20px 24px;
  overflow-y: auto;
}
.fade-enter-active, .fade-leave-active { transition: opacity 0.15s; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
