<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="虚拟机列表">
      <template #extra>
        <a-space>
          <a-input-search v-model="searchKey" placeholder="搜索虚拟机" style="width:180px" size="small" allow-clear />
          <a-select v-model="stateFilter" placeholder="全部状态" style="width:120px" size="small" allow-clear>
            <a-option value="running">运行中</a-option>
            <a-option value="shutoff">已关机</a-option>
            <a-option value="paused">已暂停</a-option>
          </a-select>
          <a-switch v-model="autoRefresh" checked-text="自动刷新" unchecked-text="自动刷新" />
          <template v-if="selectedKeys.length">
            <a-button size="small" type="primary" @click="doBatch('start')">批量启动</a-button>
            <a-button size="small" status="warning" @click="doBatch('shutdown')">批量关机</a-button>
            <a-popconfirm content="确认批量删除？" @ok="doBatch('delete')">
              <a-button size="small" status="danger">批量删除</a-button>
            </a-popconfirm>
          </template>
          <a-button @click="showImport = true">导入</a-button>
          <a-button type="primary" @click="openCreate">
            <template #icon><icon-plus /></template>
            创建虚拟机
          </a-button>
        </a-space>
      </template>
      <a-table :data="filteredVMs" :loading="loading" row-key="name" :pagination="false" :row-selection="{ type: 'checkbox', showCheckedAll: true }" v-model:selectedKeys="selectedKeys">
        <template #columns>
          <a-table-column title="名称" data-index="name">
            <template #cell="{ record }">
              <a-link @click="router.push({ name: 'vm-detail', params: { name: record.name } })">{{ record.name }}</a-link>
            </template>
          </a-table-column>
          <a-table-column title="状态" data-index="state">
            <template #cell="{ record }">
              <a-badge :status="stateBadge(record.state, record.name)" :text="stateText(record.state, record.name)" />
            </template>
          </a-table-column>
          <a-table-column title="CPU" :width="160">
            <template #cell="{ record }">
              <div>{{ record.cpu }} 核</div>
              <a-progress v-if="record.state === 'running'" :percent="record.cpu_usage / 100" :stroke-width="4" :show-text="false" style="width:80px;margin-top:2px" />
              <span v-if="record.state === 'running'" style="font-size:11px;color:#8e8e93;margin-left:4px">{{ record.cpu_usage }}%</span>
            </template>
          </a-table-column>
          <a-table-column title="内存" :width="180">
            <template #cell="{ record }">
              <div>{{ record.memory >= 1024 ? (record.memory / 1024).toFixed(1) + ' GB' : record.memory + ' MB' }}</div>
              <template v-if="record.state === 'running' && record.mem_used > 0">
                <a-progress :percent="record.mem_used / record.memory" :stroke-width="4" :show-text="false" color="#FF9500" style="width:80px;margin-top:2px" />
                <span style="font-size:11px;color:#8e8e93;margin-left:4px">{{ record.mem_used >= 1024 ? (record.mem_used / 1024).toFixed(1) + 'G' : record.mem_used + 'M' }}</span>
              </template>
            </template>
          </a-table-column>
          <a-table-column title="自动启动" :width="100">
            <template #cell="{ record }">
              <a-switch v-model="autostartMap[record.name]" size="small" @change="(v: boolean) => toggleAutostart(record.name, v)" />
            </template>
          </a-table-column>
          <a-table-column title="操作">
            <template #cell="{ record }">
              <a-space>
                <a-button v-if="record.state === 'running'" size="small" @click="openVNC(record.name)">控制台</a-button>
                <a-button v-if="record.state !== 'running' && record.state !== 'paused'" size="small" type="primary" @click="doAction(record.name, 'start')">启动</a-button>
                <a-button v-if="record.state === 'running'" size="small" status="warning" @click="doAction(record.name, 'shutdown')">关机</a-button>
                <a-popconfirm v-if="record.state === 'running'" content="强制关机会立即断电，可能丢失数据" @ok="doAction(record.name, 'destroy')">
                  <a-button size="small" status="danger">强制关机</a-button>
                </a-popconfirm>
                <a-button v-if="record.state === 'running'" size="small" @click="doAction(record.name, 'reboot')">重启</a-button>
                <a-button v-if="record.state === 'running'" size="small" @click="doAction(record.name, 'suspend')">暂停</a-button>
                <a-button v-if="record.state === 'paused'" size="small" type="primary" @click="doAction(record.name, 'resume')">恢复</a-button>
                <a-button v-if="record.state === 'shutoff'" size="small" @click="openEdit(record)">编辑</a-button>
                <a-button v-if="record.state === 'shutoff'" size="small" @click="openRename(record.name)">重命名</a-button>
                <a-button v-if="record.state === 'shutoff'" size="small" @click="openClone(record.name)">克隆</a-button>
                <a-popconfirm content="确认删除？" @ok="doDelete(record.name)">
                  <a-button size="small" status="danger">删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 创建虚拟机 -->
    <a-modal v-model:visible="showCreate" :width="680" :footer="false" :mask-closable="false" unmount-on-close>
      <template #title>
        <span style="font-size:16px;font-weight:600">创建虚拟机</span>
      </template>
      <a-steps :current="step" size="small" style="margin-bottom:24px">
        <a-step description="选择系统类型">系统</a-step>
        <a-step description="配置硬件资源">硬件</a-step>
        <a-step description="高级虚拟化选项">高级</a-step>
      </a-steps>

      <!-- Step 1: 系统类型 -->
      <div v-show="step === 1">
        <div style="margin-bottom:16px;color:var(--color-text-2);font-size:13px">选择系统类型，自动配置磁盘和网卡</div>
        <a-row :gutter="12">
          <a-col :span="8" v-for="preset in osPresets" :key="preset.key">
            <div
              class="os-card"
              :class="{ active: form.osType === preset.key }"
              @click="selectOS(preset.key)"
            >
              <div class="os-name">{{ preset.name }}</div>
              <div class="os-desc">{{ preset.desc }}</div>
              <div class="os-detail">
                <a-tag size="small" color="arcoblue">{{ preset.disk }}</a-tag>
                <a-tag size="small" color="green">{{ preset.net }}</a-tag>
              </div>
            </div>
          </a-col>
        </a-row>
        <a-form :model="form" layout="vertical" style="margin-top:20px">
          <a-form-item label="虚拟机名称" required>
            <a-input v-model="form.name" placeholder="vm-01（仅支持英文、数字、._-）" allow-clear />
          </a-form-item>
          <a-form-item label="安装镜像 (ISO)">
            <a-select v-model="form.iso" placeholder="可选，选择 ISO 镜像安装系统" allow-clear>
              <a-option v-for="iso in createISOs" :key="iso.path" :value="iso.path">{{ iso.name }}</a-option>
            </a-select>
          </a-form-item>
        </a-form>
      </div>

      <!-- Step 2: 硬件配置 -->
      <div v-show="step === 2">
        <a-form :model="form" layout="vertical">
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="CPU (核)">
                <a-input-number v-model="form.cpu" :min="1" :max="64" style="width:100%" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="内存 (MB)">
                <a-input-number v-model="form.memory" :min="256" :step="256" style="width:100%" />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="磁盘 (GB)">
                <a-input-number v-model="form.disk" :min="1" style="width:100%" />
              </a-form-item>
            </a-col>
          </a-row>
          <div style="margin-bottom:12px">
            <span style="color:var(--color-text-3);font-size:12px">快捷配置：</span>
            <a-space style="margin-left:8px">
              <a-tag v-for="q in quickSpecs" :key="q.label" checkable :checked="isQuickSpec(q)" @check="applyQuickSpec(q)" style="cursor:pointer">{{ q.label }}</a-tag>
            </a-space>
          </div>
        </a-form>
      </div>

      <!-- Step 3: 高级选项 -->
      <div v-show="step === 3">
        <a-form :model="form" layout="vertical">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="磁盘总线">
                <a-select v-model="form.diskBus">
                  <a-option value="">跟随预设 ({{ defaultBus }})</a-option>
                  <a-option value="virtio">VirtIO</a-option>
                  <a-option value="scsi">SCSI</a-option>
                  <a-option value="sata">SATA</a-option>
                  <a-option value="ide">IDE</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="网卡">
                <a-select v-model="form.netModel">
                  <a-option value="">跟随预设 ({{ defaultNet }})</a-option>
                  <a-option value="virtio">VirtIO</a-option>
                  <a-option value="e1000">E1000</a-option>
                  <a-option value="rtl8139">RTL8139</a-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="8">
              <a-form-item label="芯片组">
                <a-select v-model="form.machine">
                  <a-option value="">跟随预设 ({{ defaultMachine }})</a-option>
                  <a-option value="q35">Q35</a-option>
                  <a-option value="i440fx">i440FX</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="CPU 模式">
                <a-select v-model="form.cpuModel">
                  <a-option value="">跟随预设 ({{ defaultCpu }})</a-option>
                  <a-option value="host-passthrough">host-passthrough</a-option>
                  <a-option value="host-model">host-model</a-option>
                  <a-option value="qemu64">qemu64</a-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="时钟">
                <a-select v-model="form.clock">
                  <a-option value="">跟随预设 ({{ defaultClock }})</a-option>
                  <a-option value="utc">UTC</a-option>
                  <a-option value="localtime">本地时间</a-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="VirtIO 驱动 ISO（第二光驱）">
            <a-select v-model="form.virtioISO" placeholder="可选" allow-clear>
              <a-option v-for="iso in createISOs" :key="iso.path" :value="iso.path">{{ iso.name }}</a-option>
            </a-select>
          </a-form-item>
          <a-form-item label="网络模式">
            <a-select v-model="form.netMode">
              <a-option value="nat">NAT（默认网络）</a-option>
              <a-option value="bridge">桥接</a-option>
              <a-option value="macvtap">macvtap（直连局域网）</a-option>
            </a-select>
          </a-form-item>
          <a-form-item v-if="form.netMode === 'bridge'" label="网桥名称">
            <a-input v-model="form.bridgeName" placeholder="br0" />
          </a-form-item>
          <a-form-item v-if="form.netMode === 'macvtap'" label="物理网卡">
            <a-select v-model="form.macvtapDev" placeholder="选择物理网卡">
              <a-option v-for="nic in hostNICs" :key="nic.name" :value="nic.name">{{ nic.name }}{{ nic.ip ? ' (' + nic.ip + ')' : '' }}</a-option>
            </a-select>
          </a-form-item>
        </a-form>
      </div>

      <!-- 底部按钮 -->
      <a-divider style="margin:20px 0 16px" />
      <div style="display:flex;justify-content:space-between">
        <a-button v-if="step > 1" @click="step--">上一步</a-button>
        <div v-else />
        <a-space>
          <a-button @click="showCreate = false">取消</a-button>
          <a-button v-if="step < 3" type="primary" @click="nextStep">下一步</a-button>
          <a-button v-else type="primary" :loading="creating" @click="onCreate">
            <template #icon><icon-check /></template>
            创建
          </a-button>
        </a-space>
      </div>
    </a-modal>

    <!-- 编辑 -->
    <a-modal v-model:visible="showEdit" title="编辑虚拟机" @ok="onEdit" :ok-loading="editing">
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="CPU (核)">
          <a-input-number v-model="editForm.cpu" :min="1" :max="64" />
        </a-form-item>
        <a-form-item label="内存 (MB)">
          <a-input-number v-model="editForm.memory" :min="256" :step="256" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 克隆 -->
    <a-modal v-model:visible="showClone" title="克隆虚拟机" @ok="onClone" :ok-loading="cloning">
      <a-form :model="cloneForm" layout="vertical">
        <a-form-item label="新虚拟机名称" required>
          <a-input v-model="cloneForm.newName" :placeholder="cloneForm.srcName + '-clone'" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 重命名 -->
    <a-modal v-model:visible="showRename" title="重命名虚拟机" @ok="onRename" :ok-loading="renaming">
      <a-form :model="renameForm" layout="vertical">
        <a-form-item label="新名称" required>
          <a-input v-model="renameForm.newName" placeholder="新虚拟机名称" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 导入 -->
    <a-modal v-model:visible="showImport" title="导入虚拟机" @ok="onImport" :ok-loading="importing">
      <a-form :model="importForm" layout="vertical">
        <a-form-item label="虚拟机名称" required>
          <a-input v-model="importForm.name" placeholder="vm-imported" />
        </a-form-item>
        <a-form-item label="磁盘镜像路径" required>
          <a-input v-model="importForm.diskPath" placeholder="/var/lib/libvirt/images/disk.qcow2" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="CPU (核)">
              <a-input-number v-model="importForm.cpu" :min="1" :max="64" style="width:100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="内存 (MB)">
              <a-input-number v-model="importForm.memory" :min="256" :step="256" style="width:100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="磁盘总线">
              <a-select v-model="importForm.diskBus">
                <a-option value="virtio">VirtIO</a-option>
                <a-option value="sata">SATA</a-option>
                <a-option value="ide">IDE</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { vmApi, type VM } from '../../api/vm'
import { hostApi } from '../../api/host'
import { isoApi, type ISOFile } from '../../api/iso'
import { Message } from '@arco-design/web-vue'
import { IconPlus, IconCheck } from '@arco-design/web-vue/es/icon'

const router = useRouter()

const vms = ref<VM[]>([])
const loading = ref(false)
const autoRefresh = ref(false)
const searchKey = ref('')
const stateFilter = ref<string | undefined>(undefined)
let timer: ReturnType<typeof setInterval> | null = null

const filteredVMs = computed(() => {
  let list = vms.value
  if (searchKey.value) {
    const key = searchKey.value.toLowerCase()
    list = list.filter(v => v.name.toLowerCase().includes(key))
  }
  if (stateFilter.value) list = list.filter(v => v.state === stateFilter.value)
  return list
})

const showCreate = ref(false)
const creating = ref(false)
const step = ref(1)
const form = reactive({
  name: '', cpu: 2, memory: 2048, disk: 20, iso: '',
  osType: 'linux', diskBus: '', netModel: '',
  machine: '', cpuModel: '', clock: '', virtioISO: '',
  netMode: 'nat', bridgeName: 'br0', macvtapDev: 'eth0',
})

const createISOs = ref<ISOFile[]>([])
const hostNICs = ref<{ name: string; mac: string; ip: string; up: boolean }[]>([])

const showEdit = ref(false)
const editing = ref(false)
const editForm = reactive({ name: '', cpu: 2, memory: 2048 })

const showClone = ref(false)
const cloning = ref(false)
const cloneForm = reactive({ srcName: '', newName: '' })
const selectedKeys = ref<string[]>([])
const showRename = ref(false)
const renaming = ref(false)
const renameForm = reactive({ oldName: '', newName: '' })
const showImport = ref(false)
const importing = ref(false)
const importForm = reactive({ name: '', diskPath: '', cpu: 2, memory: 2048, diskBus: 'virtio' })
const autostartMap = ref<Record<string, boolean>>({})

const loadAutostarts = async () => {
  const map: Record<string, boolean> = {}
  await Promise.all(vms.value.map(async (vm) => {
    try { const r = await vmApi.getAutostart(vm.name); map[vm.name] = r.autostart } catch { map[vm.name] = false }
  }))
  autostartMap.value = map
}
const toggleAutostart = async (name: string, v: boolean) => {
  try { await vmApi.setAutostart(name, v); Message.success(v ? '已开启自动启动' : '已关闭自动启动') } catch { Message.error('设置失败'); autostartMap.value[name] = !v }
}

// OS presets
const osPresets = [
  { key: 'linux', name: 'Linux', desc: '半虚拟化，性能最佳', disk: 'VirtIO', net: 'VirtIO' },
  { key: 'windows', name: 'Windows', desc: '全虚拟化，兼容性好', disk: 'SATA', net: 'E1000' },
  { key: 'legacy', name: '兼容', desc: '最大兼容性', disk: 'IDE', net: 'RTL8139' },
]

const presetDefaults: Record<string, { bus: string; net: string; machine: string; cpu: string; clock: string }> = {
  linux: { bus: 'virtio', net: 'virtio', machine: 'i440fx', cpu: '默认', clock: 'utc' },
  windows: { bus: 'sata', net: 'e1000', machine: 'q35', cpu: 'host-passthrough', clock: 'localtime' },
  legacy: { bus: 'ide', net: 'rtl8139', machine: 'i440fx', cpu: '默认', clock: 'utc' },
}

const defaultBus = computed(() => presetDefaults[form.osType]?.bus || 'virtio')
const defaultNet = computed(() => presetDefaults[form.osType]?.net || 'virtio')
const defaultMachine = computed(() => presetDefaults[form.osType]?.machine || 'i440fx')
const defaultCpu = computed(() => presetDefaults[form.osType]?.cpu || '默认')
const defaultClock = computed(() => presetDefaults[form.osType]?.clock || 'utc')
const currentPresetDesc = computed(() => {
  const p = osPresets.find(o => o.key === form.osType)
  return p ? `${p.name} — ${p.desc}` : ''
})

// Quick hardware specs
const quickSpecs = [
  { label: '轻量 1C1G', cpu: 1, memory: 1024, disk: 10 },
  { label: '标准 2C2G', cpu: 2, memory: 2048, disk: 20 },
  { label: '增强 4C4G', cpu: 4, memory: 4096, disk: 40 },
  { label: '高配 8C8G', cpu: 8, memory: 8192, disk: 80 },
]

const isQuickSpec = (q: typeof quickSpecs[0]) => form.cpu === q.cpu && form.memory === q.memory && form.disk === q.disk
const applyQuickSpec = (q: typeof quickSpecs[0]) => { form.cpu = q.cpu; form.memory = q.memory; form.disk = q.disk }

const selectOS = (key: string) => {
  form.osType = key
  form.diskBus = ''; form.netModel = ''
  form.machine = ''; form.cpuModel = ''; form.clock = ''
}

const stateText = (s: string, name?: string) => {
  if (name && pendingStates.value[name]) return pendingStates.value[name]
  return ({ running: '运行中', shutoff: '已关机', paused: '已暂停', shutdown: '关机中' }[s] || s)
}

const stateBadge = (s: string, name?: string) => {
  if (name && pendingStates.value[name]) return 'processing' as any
  return ({ running: 'success', paused: 'warning', shutoff: 'default' }[s] || 'default') as any
}

const loadVMs = async () => {
  loading.value = true
  try { vms.value = await vmApi.list(); loadAutostarts() } catch { /* */ }
  loading.value = false
}

watch(autoRefresh, (v) => {
  if (timer) { clearInterval(timer); timer = null }
  if (v) timer = setInterval(loadVMs, 5000)
})

const actionTips: Record<string, string> = {
  shutdown: '关机信号已发送，需要虚拟机内操作系统支持 ACPI，未安装系统的虚拟机请使用强制关机',
  reboot: '重启信号已发送，需要虚拟机内操作系统支持 ACPI',
  destroy: '已强制关机',
}

const openVNC = (name: string) => {
  const route = router.resolve({ name: 'vnc', params: { name } })
  window.open(route.href, '_blank')
}

const pendingStates = ref<Record<string, string>>({})

const doAction = async (name: string, action: 'start' | 'shutdown' | 'destroy' | 'reboot' | 'suspend' | 'resume') => {
  try {
    await vmApi[action](name)
    Message.success(actionTips[action] || '操作成功')
    const transient: Record<string, string> = { start: '启动中', shutdown: '关机中', destroy: '关机中', reboot: '重启中', suspend: '暂停中', resume: '恢复中' }
    pendingStates.value[name] = transient[action] || '操作中'
    // 轮询直到状态变化
    let tries = 0
    const poll = setInterval(async () => {
      tries++
      await loadVMs()
      const vm = vms.value.find(v => v.name === name)
      const done = !vm || (action === 'shutdown' && vm.state === 'shutoff') || (action === 'destroy' && vm.state === 'shutoff') || (action === 'start' && vm.state === 'running') || (action === 'reboot' && tries > 2) || (action === 'suspend' && vm.state === 'paused') || (action === 'resume' && vm.state === 'running')
      if (done || tries >= 20) { clearInterval(poll); delete pendingStates.value[name] }
    }, 2000)
  } catch { Message.error('操作失败') }
}

const doDelete = async (name: string) => {
  try {
    await vmApi.delete(name)
    Message.success('已删除')
    loadVMs()
  } catch { Message.error('删除失败') }
}

const openCreate = async () => {
  step.value = 1
  Object.assign(form, { name: '', cpu: 2, memory: 2048, disk: 20, iso: '', osType: 'linux', diskBus: '', netModel: '', virtioISO: '', netMode: 'nat', bridgeName: 'br0', macvtapDev: 'eth0' })
  try { createISOs.value = await isoApi.list() } catch { /* */ }
  try { hostNICs.value = await hostApi.nics(); if (hostNICs.value.length) form.macvtapDev = hostNICs.value[0].name } catch { /* */ }
  showCreate.value = true
}

const nextStep = () => {
  if (step.value === 1) {
    if (!form.name.trim()) { Message.warning('请输入虚拟机名称'); return }
    if (!/^[a-zA-Z0-9._-]+$/.test(form.name)) { Message.warning('名称仅支持英文字母、数字、._-'); return }
  }
  step.value++
}

const onCreate = async () => {
  creating.value = true
  try {
    await vmApi.create({
      name: form.name, cpu: form.cpu, memory: form.memory, disk: form.disk,
      os_type: form.osType,
      iso: form.iso || undefined,
      disk_bus: form.diskBus || undefined,
      net_model: form.netModel || undefined,
      machine: form.machine || undefined,
      cpu_model: form.cpuModel || undefined,
      clock: form.clock || undefined,
      virtio_iso: form.virtioISO || undefined,
      net_mode: form.netMode || undefined,
      bridge_name: form.netMode === 'bridge' ? form.bridgeName : undefined,
      macvtap_dev: form.netMode === 'macvtap' ? form.macvtapDev : undefined,
    })
    Message.success('创建成功')
    showCreate.value = false
    loadVMs()
  } catch { Message.error('创建失败') }
  creating.value = false
}

const openEdit = (vm: VM) => {
  editForm.name = vm.name; editForm.cpu = vm.cpu; editForm.memory = vm.memory
  showEdit.value = true
}

const onEdit = async () => {
  editing.value = true
  try {
    await vmApi.update(editForm.name, { cpu: editForm.cpu, memory: editForm.memory })
    Message.success('修改成功'); showEdit.value = false; loadVMs()
  } catch { Message.error('修改失败') }
  editing.value = false
}

const openClone = (name: string) => {
  cloneForm.srcName = name; cloneForm.newName = name + '-clone'
  showClone.value = true
}

const onClone = async () => {
  cloning.value = true
  try {
    await vmApi.clone(cloneForm.srcName, cloneForm.newName)
    Message.success('克隆成功'); showClone.value = false; loadVMs()
  } catch { Message.error('克隆失败') }
  cloning.value = false
}

const openRename = (name: string) => {
  renameForm.oldName = name; renameForm.newName = name
  showRename.value = true
}
const onRename = async () => {
  renaming.value = true
  try {
    await vmApi.rename(renameForm.oldName, renameForm.newName)
    Message.success('重命名成功'); showRename.value = false; loadVMs()
  } catch { Message.error('重命名失败') }
  renaming.value = false
}

const onImport = async () => {
  if (!importForm.name || !importForm.diskPath) { Message.warning('请填写名称和磁盘路径'); return }
  importing.value = true
  try {
    await vmApi.import({ name: importForm.name, disk_path: importForm.diskPath, cpu: importForm.cpu, memory: importForm.memory, disk_bus: importForm.diskBus })
    Message.success('导入成功'); showImport.value = false; Object.assign(importForm, { name: '', diskPath: '', cpu: 2, memory: 2048, diskBus: 'virtio' }); loadVMs()
  } catch { Message.error('导入失败') }
  importing.value = false
}

const doBatch = async (action: string) => {
  try {
    await vmApi.batch(selectedKeys.value, action)
    Message.success('操作完成'); selectedKeys.value = []; loadVMs()
  } catch { Message.error('操作失败') }
}

onMounted(loadVMs)
onBeforeUnmount(() => { if (timer) clearInterval(timer) })
</script>

<style scoped>
.os-card {
  border: 1.5px solid rgba(0,0,0,0.08);
  border-radius: 10px;
  padding: 18px 14px;
  text-align: center;
  cursor: pointer;
  transition: all 0.15s;
  user-select: none;
}
.os-card:hover {
  border-color: #007AFF;
}
.os-card.active {
  border-color: #007AFF;
  background: rgba(0,122,255,0.04);
}
.os-name {
  font-size: 14px;
  font-weight: 600;
  color: #1c1c1e;
}
.os-desc {
  font-size: 11px;
  color: #8e8e93;
  margin: 3px 0 8px;
}
.os-detail {
  display: flex;
  justify-content: center;
  gap: 4px;
}
</style>
