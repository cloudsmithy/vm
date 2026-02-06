<template>
  <div>
    <a-row :gutter="14">
      <a-col :span="6" v-for="item in stats" :key="item.title">
        <div class="stat-card">
          <div class="stat-value">{{ item.value }}<span class="stat-unit">{{ item.suffix }}</span></div>
          <div class="stat-label">{{ item.title }}</div>
        </div>
      </a-col>
    </a-row>

    <a-row :gutter="14" style="margin-top:14px">
      <a-col :span="12">
        <a-card>
          <template #title>内存</template>
          <div class="ring-wrap">
            <a-progress type="circle" :percent="memPercent" :stroke-width="7" :width="140" color="#007AFF">
              <template #text="{ percent }">
                <div class="ring-text">
                  <div class="ring-num">{{ percent }}%</div>
                  <div class="ring-sub">{{ memUsed }} / {{ memTotal }} GB</div>
                </div>
              </template>
            </a-progress>
          </div>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card>
          <template #title>虚拟机</template>
          <div class="ring-wrap">
            <a-progress type="circle" :percent="vmPercent" :stroke-width="7" :width="140" color="#34C759">
              <template #text>
                <div class="ring-text">
                  <div class="ring-num">{{ host?.vm_running || 0 }}</div>
                  <div class="ring-sub">运行中 / {{ host?.vm_total || 0 }} 台</div>
                </div>
              </template>
            </a-progress>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <a-card style="margin-top:14px">
      <template #title>主机</template>
      <a-descriptions :column="2" bordered v-if="host" size="large">
        <a-descriptions-item label="主机名">{{ host.hostname }}</a-descriptions-item>
        <a-descriptions-item label="CPU">{{ host.cpu_model }}</a-descriptions-item>
        <a-descriptions-item label="核数">{{ host.cpu_count }}</a-descriptions-item>
        <a-descriptions-item label="内存">{{ (host.memory_total / 1024).toFixed(1) }} GB</a-descriptions-item>
      </a-descriptions>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { hostApi, type HostInfo } from '../../api/host'

const host = ref<HostInfo | null>(null)

const stats = computed(() => [
  { title: '虚拟机', value: host.value?.vm_total ?? 0, suffix: '台' },
  { title: '运行中', value: host.value?.vm_running ?? 0, suffix: '台' },
  { title: 'CPU', value: host.value?.cpu_count ?? 0, suffix: '核' },
  { title: '可用内存', value: host.value ? +(host.value.memory_free / 1024).toFixed(1) : 0, suffix: 'GB' },
])

const memTotal = computed(() => host.value ? (host.value.memory_total / 1024).toFixed(1) : '0')
const memUsed = computed(() => host.value ? ((host.value.memory_total - host.value.memory_free) / 1024).toFixed(1) : '0')
const memPercent = computed(() => {
  if (!host.value || !host.value.memory_total) return 0
  return Math.round(((host.value.memory_total - host.value.memory_free) / host.value.memory_total) * 100)
})
const vmPercent = computed(() => {
  if (!host.value || !host.value.vm_total) return 0
  return Math.round((host.value.vm_running / host.value.vm_total) * 100)
})

const load = async () => { try { host.value = await hostApi.info() } catch {} }
let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => { load(); timer = setInterval(load, 10000) })
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.stat-card {
  background: #fff;
  border-radius: 10px;
  padding: 16px 18px;
  box-shadow: 0 0.5px 1px rgba(0,0,0,0.05), 0 1px 3px rgba(0,0,0,0.04);
}
.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: #1c1c1e;
  letter-spacing: -0.5px;
  line-height: 1;
}
.stat-unit { font-size: 13px; font-weight: 500; color: #8e8e93; margin-left: 2px; }
.stat-label { font-size: 12px; color: #8e8e93; margin-top: 4px; }
.ring-wrap { display: flex; justify-content: center; padding: 20px 0; }
.ring-text { text-align: center; }
.ring-num { font-size: 26px; font-weight: 600; color: #1c1c1e; letter-spacing: -0.5px; }
.ring-sub { font-size: 11px; color: #8e8e93; margin-top: 2px; }
</style>
