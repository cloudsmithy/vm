<template>
  <div class="dash">
    <!-- 顶部概览卡片 -->
    <div class="stat-row">
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(0,122,255,0.1);color:#007AFF">
          <icon-computer />
        </div>
        <div class="stat-body">
          <div class="stat-label">主机名</div>
          <div class="stat-value">{{ host?.hostname ?? '-' }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(52,199,89,0.1);color:#34C759">
          <icon-clock-circle />
        </div>
        <div class="stat-body">
          <div class="stat-label">运行时间</div>
          <div class="stat-value">{{ uptimeStr }}</div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(255,149,0,0.1);color:#FF9500">
          <icon-desktop />
        </div>
        <div class="stat-body">
          <div class="stat-label">虚拟机</div>
          <div class="stat-value">{{ host?.vm_running ?? 0 }} <span class="stat-sub">/ {{ host?.vm_total ?? 0 }} 台</span></div>
        </div>
      </div>
      <div class="stat-card">
        <div class="stat-icon" style="background:rgba(175,82,222,0.1);color:#AF52DE">
          <icon-thunderbolt />
        </div>
        <div class="stat-body">
          <div class="stat-label">系统负载</div>
          <div class="stat-value">{{ host?.load_avg?.[0]?.toFixed(2) ?? '-' }} <span class="stat-sub">{{ host?.load_avg?.[1]?.toFixed(2) }} {{ host?.load_avg?.[2]?.toFixed(2) }}</span></div>
        </div>
      </div>
    </div>

    <!-- CPU 和内存环形图 + 趋势 -->
    <div class="chart-row">
      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-title">CPU 使用率</span>
          <span class="gauge-pct" :style="{ color: cpuColor }">{{ host?.cpu_usage ?? 0 }}%</span>
        </div>
        <v-chart :option="cpuGaugeOpt" autoresize style="height:180px" />
        <div class="gauge-info">
          <span>{{ host?.cpu_model ?? '-' }}</span>
          <span>{{ host?.cpu_count ?? 0 }} 核</span>
        </div>
      </div>
      <div class="gauge-card">
        <div class="gauge-header">
          <span class="gauge-title">内存使用率</span>
          <span class="gauge-pct" :style="{ color: memColor }">{{ memPercent }}%</span>
        </div>
        <v-chart :option="memGaugeOpt" autoresize style="height:180px" />
        <div class="gauge-info">
          <span>已用 {{ memUsed }} GB</span>
          <span>共 {{ memTotal }} GB</span>
        </div>
      </div>
      <div class="trend-card">
        <div class="gauge-header">
          <span class="gauge-title">CPU / 内存趋势</span>
          <span class="gauge-pct" style="font-size:11px;color:#8e8e93">最近 {{ historyLen }} 次采样</span>
        </div>
        <v-chart :option="trendOpt" autoresize style="height:210px" />
      </div>
    </div>

    <!-- 磁盘使用 -->
    <div class="disk-section">
      <div class="section-title">磁盘使用</div>
      <div class="disk-row">
        <div class="disk-card" v-for="d in host?.disks ?? []" :key="d.mount">
          <div class="disk-header">
            <icon-storage class="disk-icon" />
            <div>
              <div class="disk-mount">{{ d.mount }}</div>
              <div class="disk-device">{{ d.device }}</div>
            </div>
          </div>
          <div class="disk-bar-wrap">
            <div class="disk-bar" :style="{ width: d.percent + '%', background: diskColor(d.percent) }"></div>
          </div>
          <div class="disk-stats">
            <span>{{ d.used }}G / {{ d.total }}G</span>
            <span :style="{ color: diskColor(d.percent), fontWeight: 600 }">{{ d.percent }}%</span>
          </div>
        </div>
      </div>
    </div>

    <!-- 虚拟机资源排行 -->
    <div class="info-section" v-if="runningVMs.length">
      <div class="section-title">虚拟机资源占用</div>
      <div class="info-card">
        <div class="vm-rank-row vm-rank-header">
          <span class="vm-rank-name">名称</span>
          <span class="vm-rank-bar">CPU</span>
          <span class="vm-rank-bar">内存</span>
        </div>
        <div class="vm-rank-row" v-for="vm in runningVMs" :key="vm.name">
          <span class="vm-rank-name">{{ vm.name }}</span>
          <span class="vm-rank-bar">
            <div class="mini-bar-wrap"><div class="mini-bar" :style="{ width: vm.cpu_usage + '%', background: getColor(vm.cpu_usage) }"></div></div>
            <span class="vm-rank-pct">{{ vm.cpu_usage }}%</span>
          </span>
          <span class="vm-rank-bar">
            <div class="mini-bar-wrap"><div class="mini-bar" :style="{ width: (vm.mem_used && vm.memory ? vm.mem_used / vm.memory * 100 : 0) + '%', background: '#FF9500' }"></div></div>
            <span class="vm-rank-pct">{{ vm.mem_used >= 1024 ? (vm.mem_used / 1024).toFixed(1) + 'G' : vm.mem_used + 'M' }} / {{ vm.memory >= 1024 ? (vm.memory / 1024).toFixed(1) + 'G' : vm.memory + 'M' }}</span>
          </span>
        </div>
      </div>
    </div>

    <!-- 系统信息 -->
    <div class="info-section">
      <div class="section-title">系统信息</div>
      <div class="info-card">
        <div class="info-row" v-for="item in sysInfo" :key="item.label">
          <span class="info-label">{{ item.label }}</span>
          <span class="info-value">{{ item.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { hostApi, type HostInfo } from '../../api/host'
import { vmApi, type VM } from '../../api/vm'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { GaugeChart, LineChart } from 'echarts/charts'
import { CanvasRenderer } from 'echarts/renderers'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import {
  IconComputer, IconClockCircle, IconDesktop, IconThunderbolt, IconStorage,
} from '@arco-design/web-vue/es/icon'

use([CanvasRenderer, GaugeChart, LineChart, GridComponent, TooltipComponent, LegendComponent])

const host = ref<HostInfo | null>(null)
const allVMs = ref<VM[]>([])
const runningVMs = computed(() => allVMs.value.filter(v => v.state === 'running').sort((a, b) => b.cpu_usage - a.cpu_usage))
const cpuHistory = ref<number[]>([])
const memHistory = ref<number[]>([])
const timeLabels = ref<string[]>([])
const MAX_HISTORY = 60
const historyLen = computed(() => cpuHistory.value.length)

const memTotal = computed(() => host.value ? (host.value.memory_total / 1024).toFixed(1) : '0')
const memUsed = computed(() => host.value ? ((host.value.memory_total - host.value.memory_free) / 1024).toFixed(1) : '0')
const memPercent = computed(() => {
  if (!host.value || !host.value.memory_total) return 0
  return Math.round(((host.value.memory_total - host.value.memory_free) / host.value.memory_total) * 100)
})

const uptimeStr = computed(() => {
  if (!host.value) return '-'
  const s = host.value.uptime
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return `${d}天 ${h}小时`
  if (h > 0) return `${h}小时 ${m}分钟`
  return `${m}分钟`
})

const getColor = (v: number) => v > 80 ? '#FF3B30' : v > 60 ? '#FF9500' : '#34C759'
const cpuColor = computed(() => getColor(host.value?.cpu_usage ?? 0))
const memColor = computed(() => getColor(memPercent.value))
const diskColor = (p: number) => p > 90 ? '#FF3B30' : p > 70 ? '#FF9500' : '#007AFF'

const makeGauge = (value: number, color: string) => ({
  series: [{
    type: 'gauge',
    startAngle: 220,
    endAngle: -40,
    radius: '90%',
    progress: { show: true, width: 14, roundCap: true, itemStyle: { color } },
    axisLine: { lineStyle: { width: 14, color: [[1, 'rgba(0,0,0,0.06)']] } },
    axisTick: { show: false },
    splitLine: { show: false },
    axisLabel: { show: false },
    pointer: { show: false },
    title: { show: false },
    detail: { show: false },
    data: [{ value }],
  }],
})

const cpuGaugeOpt = computed(() => makeGauge(host.value?.cpu_usage ?? 0, cpuColor.value))
const memGaugeOpt = computed(() => makeGauge(memPercent.value, memColor.value))

const trendOpt = computed(() => ({
  tooltip: { trigger: 'axis', backgroundColor: 'rgba(255,255,255,0.95)', borderColor: 'rgba(0,0,0,0.08)', textStyle: { fontSize: 12, color: '#1c1c1e' } },
  legend: { data: ['CPU', '内存'], right: 0, top: 0, textStyle: { fontSize: 11, color: '#8e8e93' }, itemWidth: 12, itemHeight: 3 },
  grid: { left: 36, right: 12, top: 30, bottom: 24 },
  xAxis: { type: 'category', data: timeLabels.value, axisLabel: { fontSize: 10, color: '#aeaeb2' }, axisLine: { lineStyle: { color: 'rgba(0,0,0,0.06)' } }, axisTick: { show: false } },
  yAxis: { type: 'value', min: 0, max: 100, axisLabel: { fontSize: 10, color: '#aeaeb2', formatter: '{value}%' }, splitLine: { lineStyle: { color: 'rgba(0,0,0,0.04)' } } },
  series: [
    { name: 'CPU', type: 'line', data: cpuHistory.value, smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#007AFF' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(0,122,255,0.15)' }, { offset: 1, color: 'rgba(0,122,255,0)' }] } } },
    { name: '内存', type: 'line', data: memHistory.value, smooth: true, symbol: 'none', lineStyle: { width: 2, color: '#FF9500' }, areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(255,149,0,0.15)' }, { offset: 1, color: 'rgba(255,149,0,0)' }] } } },
  ],
}))

const sysInfo = computed(() => {
  if (!host.value) return []
  return [
    { label: '处理器', value: host.value.cpu_model },
    { label: '核心数', value: `${host.value.cpu_count} 核` },
    { label: '总内存', value: `${memTotal.value} GB` },
    { label: '可用内存', value: `${(host.value.memory_free / 1024).toFixed(1)} GB` },
    { label: '负载均值', value: `${host.value.load_avg[0].toFixed(2)} / ${host.value.load_avg[1].toFixed(2)} / ${host.value.load_avg[2].toFixed(2)}` },
    { label: '运行时间', value: uptimeStr.value },
  ]
})

const load = async () => {
  if (!mounted) return
  try {
    host.value = await hostApi.info()
    try { allVMs.value = await vmApi.list() } catch {}
    const now = new Date()
    const label = `${now.getHours().toString().padStart(2, '0')}:${now.getMinutes().toString().padStart(2, '0')}:${now.getSeconds().toString().padStart(2, '0')}`
    cpuHistory.value.push(host.value.cpu_usage)
    memHistory.value.push(memPercent.value)
    timeLabels.value.push(label)
    if (cpuHistory.value.length > MAX_HISTORY) {
      cpuHistory.value.shift()
      memHistory.value.shift()
      timeLabels.value.shift()
    }
  } catch {}
}

let timer: ReturnType<typeof setInterval> | null = null
let mounted = true
onMounted(() => { load(); timer = setInterval(load, 3000) })
onBeforeUnmount(() => { mounted = false; if (timer) { clearInterval(timer); timer = null } })
</script>

<style scoped>
.dash { max-width: 1200px; }

/* 顶部统计卡片 */
.stat-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin-bottom: 16px;
}
.stat-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px;
  display: flex;
  align-items: center;
  gap: 14px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}
.stat-card:hover {
  box-shadow: 0 2px 8px rgba(0,0,0,0.06);
}
.stat-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  flex-shrink: 0;
}
.stat-body { min-width: 0; }
.stat-label { font-size: 12px; color: #86868b; font-weight: 600; letter-spacing: -0.1px; }
.stat-value { font-size: 18px; font-weight: 700; color: #1d1d1f; margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; letter-spacing: -0.3px; }
.stat-sub { font-size: 13px; font-weight: 400; color: #86868b; }

/* 图表行 */
.chart-row {
  display: grid;
  grid-template-columns: 1fr 1fr 2fr;
  gap: 14px;
  margin-bottom: 16px;
}
.gauge-card, .trend-card {
  background: #fff;
  border-radius: 12px;
  padding: 18px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}
.gauge-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.gauge-title { font-size: 14px; font-weight: 600; color: #1d1d1f; letter-spacing: -0.2px; }
.gauge-pct { font-size: 22px; font-weight: 700; letter-spacing: -0.5px; }
.gauge-info {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #86868b;
  margin-top: 4px;
  padding: 0 4px;
}

/* 磁盘 */
.section-title {
  font-size: 15px;
  font-weight: 700;
  color: #1d1d1f;
  margin-bottom: 12px;
  letter-spacing: -0.2px;
}
.disk-section { margin-bottom: 16px; }
.disk-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 14px;
}
.disk-card {
  background: #fff;
  border-radius: 12px;
  padding: 16px 18px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}
.disk-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.disk-icon { font-size: 20px; color: #007AFF; }
.disk-mount { font-size: 14px; font-weight: 600; color: #1d1d1f; }
.disk-device { font-size: 12px; color: #86868b; }
.disk-bar-wrap {
  height: 6px;
  background: #e8e8ed;
  border-radius: 3px;
  overflow: hidden;
}
.disk-bar {
  height: 100%;
  border-radius: 3px;
  transition: width 0.5s ease;
}
.disk-stats {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #86868b;
  margin-top: 8px;
}

/* 系统信息 */
.info-section { margin-bottom: 16px; }
.info-card {
  background: #fff;
  border-radius: 12px;
  padding: 4px 0;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}
.info-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  border-bottom: 1px solid #f5f5f7;
}
.info-row:last-child { border-bottom: none; }
.info-label { font-size: 14px; color: #1d1d1f; font-weight: 500; }
.info-value { font-size: 14px; color: #86868b; max-width: 60%; text-align: right; word-break: break-all; }

.vm-rank-row { display: flex; align-items: center; padding: 10px 20px; border-bottom: 0.5px solid rgba(0,0,0,0.04); gap: 12px; }
.vm-rank-row:last-child { border-bottom: none; }
.vm-rank-header { font-size: 11px; color: #8e8e93; font-weight: 600; text-transform: uppercase; }
.vm-rank-name { width: 140px; font-size: 13px; font-weight: 500; color: #1c1c1e; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.vm-rank-bar { flex: 1; display: flex; align-items: center; gap: 8px; }
.vm-rank-pct { font-size: 11px; color: #8e8e93; min-width: 60px; text-align: right; }
.mini-bar-wrap { flex: 1; height: 6px; background: rgba(0,0,0,0.04); border-radius: 3px; overflow: hidden; }
.mini-bar { height: 100%; border-radius: 3px; transition: width 0.5s ease; }
</style>
