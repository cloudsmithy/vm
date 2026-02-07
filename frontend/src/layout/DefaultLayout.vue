<template>
  <div class="app-layout">
    <div class="sidebar" :class="{ collapsed }">
      <div class="sidebar-logo" @click="router.push({ name: 'dashboard' })" style="cursor:pointer">
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
          <span v-if="!collapsed && item.key === 'vm' && vmBadge" class="nav-badge">{{ vmBadge }}</span>
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
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { hostApi } from '../api/host'
import {
  IconDashboard, IconDesktop, IconWifi, IconStorage, IconHistory, IconFile,
  IconLeft, IconRight, IconShareExternal, IconSwap,
} from '@arco-design/web-vue/es/icon'

const router = useRouter()
const route = useRoute()
const collapsed = ref(false)
const vmBadge = ref('')

const loadVMCount = async () => {
  try {
    const info = await hostApi.info()
    vmBadge.value = `${info.vm_running}/${info.vm_total}`
  } catch {}
}
let badgeTimer: ReturnType<typeof setInterval> | null = null
onMounted(() => { loadVMCount(); badgeTimer = setInterval(loadVMCount, 10000) })
onBeforeUnmount(() => { if (badgeTimer) clearInterval(badgeTimer) })

const menuItems = [
  { key: 'dashboard', label: '概览', icon: IconDashboard },
  { key: 'vm', label: '虚拟机', icon: IconDesktop },
  { key: 'snapshot', label: '快照', icon: IconHistory },
  { key: 'network', label: '网络', icon: IconWifi },
  { key: 'bridge', label: '网桥', icon: IconShareExternal },
  { key: 'portforward', label: '端口转发', icon: IconSwap },
  { key: 'storage', label: '存储', icon: IconStorage },
  { key: 'iso', label: '镜像', icon: IconFile },
]

const currentRoute = computed(() => {
  const name = route.name as string
  return name === 'vm-detail' ? 'vm' : name
})

const titles: Record<string, string> = {
  dashboard: '概览', vm: '虚拟机', 'vm-detail': '虚拟机详情',
  snapshot: '快照', network: '网络', bridge: '网桥', portforward: '端口转发', storage: '存储', iso: '镜像',
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
  background: #ffffff;
  display: flex;
  flex-direction: column;
  transition: width 0.2s ease;
  flex-shrink: 0;
  border-right: 1px solid #e8e8ed;
}
.sidebar.collapsed { width: 60px; }

.sidebar-logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-bottom: 1px solid #f0f0f0;
}
.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #1d1d1f;
  letter-spacing: -0.3px;
}

.sidebar-nav {
  flex: 1;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.collapsed .sidebar-nav { padding: 8px 6px; }

.nav-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.12s;
  color: #1d1d1f;
  font-size: 15px;
  white-space: nowrap;
  overflow: hidden;
}
.collapsed .nav-item { justify-content: center; padding: 10px; }

.nav-item:hover { background: #f5f5f7; }
.nav-item.active {
  background: #007AFF;
  color: #fff;
}
.nav-item.active .nav-icon { color: #fff; }

.nav-icon { font-size: 20px; flex-shrink: 0; color: #86868b; }
.nav-label { font-weight: 500; }
.nav-badge { margin-left: auto; font-size: 11px; background: #f0f0f0; color: #86868b; padding: 2px 7px; border-radius: 10px; font-weight: 600; }
.nav-item.active .nav-badge { background: rgba(255,255,255,0.25); color: #fff; }

.sidebar-bottom {
  padding: 12px;
  display: flex;
  justify-content: center;
  cursor: pointer;
  flex-shrink: 0;
  border-top: 1px solid #f0f0f0;
}
.sidebar-bottom .nav-icon { color: #aeaeb2; font-size: 14px; }

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #f5f5f7;
}
.app-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 28px;
  background: #fff;
  border-bottom: 1px solid #e8e8ed;
  flex-shrink: 0;
}
.page-title {
  font-size: 20px;
  font-weight: 700;
  color: #1d1d1f;
  margin: 0;
  letter-spacing: -0.3px;
}
.app-content {
  flex: 1;
  padding: 24px 28px;
  overflow-y: auto;
}
</style>
