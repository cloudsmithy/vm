<template>
  <div class="vnc-layout">
    <header class="vnc-header">
      <a-space>
        <a-button size="small" @click="$router.push({ name: 'vm' })">
          <template #icon><icon-left /></template>
          返回
        </a-button>
        <span class="vnc-title">{{ vmName }}</span>
        <a-badge :status="status === 'connected' ? 'success' : 'default'" :text="status === 'connected' ? '已连接' : '未连接'" />
      </a-space>
      <a-space>
        <a-button size="small" @click="sendCtrlAltDel">Ctrl+Alt+Del</a-button>
      </a-space>
    </header>
    <div class="vnc-body">
      <div ref="vncContainer" class="vnc-screen"></div>
      <div v-if="status !== 'connected'" class="vnc-overlay">
        <a-spin v-if="status === 'connecting'" tip="连接中..." />
        <a-result v-else status="error" :title="errorMsg">
          <template #extra><a-button type="primary" @click="connect">重新连接</a-button></template>
        </a-result>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRoute } from 'vue-router'
import { IconLeft } from '@arco-design/web-vue/es/icon'
import RFB from '@novnc/novnc/lib/rfb.js'

const route = useRoute()
const vmName = computed(() => route.params.name as string)
const vncContainer = ref<HTMLDivElement>()
const status = ref<'connecting' | 'connected' | 'error'>('connecting')
const errorMsg = ref('')
let rfb: any = null

const connect = () => {
  if (rfb) { rfb.disconnect(); rfb = null }
  status.value = 'connecting'; errorMsg.value = ''
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  try {
    rfb = new RFB(vncContainer.value!, `${proto}//${location.host}/ws/vnc/${vmName.value}`)
    rfb.scaleViewport = true; rfb.resizeSession = true
    rfb.addEventListener('connect', () => { status.value = 'connected' })
    rfb.addEventListener('disconnect', (e: any) => { status.value = 'error'; errorMsg.value = e.detail?.clean ? '连接已断开' : '连接异常断开' })
  } catch (e: any) { status.value = 'error'; errorMsg.value = e.message || '连接失败' }
}

const sendCtrlAltDel = () => rfb?.sendCtrlAltDel()
onMounted(connect)
onBeforeUnmount(() => rfb?.disconnect())
</script>

<style scoped>
.vnc-layout { height: 100vh; display: flex; flex-direction: column; background: #000; }
.vnc-header {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  background: #1d1d1f;
  flex-shrink: 0;
}
.vnc-header :deep(.arco-btn) { color: rgba(255,255,255,0.8); border-color: rgba(255,255,255,0.15); background: rgba(255,255,255,0.06); }
.vnc-header :deep(.arco-btn:hover) { background: rgba(255,255,255,0.12); }
.vnc-title { font-size: 14px; font-weight: 600; color: #fff; }
.vnc-body { flex: 1; position: relative; }
.vnc-screen { width: 100%; height: 100%; }
.vnc-overlay { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; background: rgba(0,0,0,0.85); }
.vnc-overlay :deep(.arco-spin-tip) { color: rgba(255,255,255,0.7); }
.vnc-overlay :deep(.arco-result-title) { color: rgba(255,255,255,0.9); }
</style>
