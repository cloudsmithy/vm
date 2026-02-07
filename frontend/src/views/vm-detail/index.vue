<template>
  <div class="vm-detail">
    <div class="detail-header animate-in">
      <a-button @click="router.push({ name: 'vm' })" style="margin-right:12px">
        <template #icon><icon-left /></template>
      </a-button>
      <div class="header-info">
        <h2 class="vm-name">{{ vmName }}</h2>
        <a-badge v-if="detail" :status="pendingState ? 'processing' : detail.state === 'running' ? 'success' : detail.state === 'paused' ? 'warning' : 'normal'" :text="pendingState || stateText(detail.state)" />
      </div>
      <div style="flex:1" />
      <a-space>
        <a-switch v-model="autostart" @change="onAutostartChange" checked-text="自动启动" unchecked-text="自动启动" size="small" />
        <a-popconfirm content="将弹出ISO并设置从硬盘启动，确认系统已安装完成？" @ok="doFinishInstall">
          <a-button v-if="hasISO" size="small" status="success">完成安装</a-button>
        </a-popconfirm>
        <a-button v-if="detail?.state !== 'running' && detail?.state !== 'paused'" size="small" type="primary" @click="doAction('start')">启动</a-button>
        <a-button v-if="detail?.state === 'running'" size="small" status="warning" @click="doAction('shutdown')">关机</a-button>
        <a-popconfirm v-if="detail?.state === 'running'" content="强制关机会立即断电，可能丢失数据">
          <template #content>强制关机会立即断电，可能丢失数据</template>
          <a-button size="small" status="danger" @click="doAction('destroy')">强制关机</a-button>
        </a-popconfirm>
        <a-button v-if="detail?.state === 'running'" size="small" @click="doAction('reboot')">重启</a-button>
        <a-button v-if="detail?.state === 'running'" size="small" @click="doAction('suspend')">暂停</a-button>
        <a-button v-if="detail?.state === 'paused'" size="small" type="primary" @click="doAction('resume')">恢复</a-button>
        <a-button v-if="detail?.state === 'running'" @click="openVNC">控制台</a-button>
        <a-button v-if="detail?.state === 'shutoff'" size="small" @click="openEdit()">编辑</a-button>
        <a-button v-if="detail?.state === 'shutoff'" size="small" @click="showClone = true">克隆</a-button>
        <a-button v-if="detail?.state === 'shutoff'" size="small" @click="showRename = true">重命名</a-button>
        <a-button @click="loadDetail">刷新</a-button>
      </a-space>
    </div>

    <a-row :gutter="16" class="animate-in" v-if="detail">
      <a-col :span="6" v-for="item in infoCards" :key="item.label">
        <div class="info-card" :class="{ clickable: item.editable }" @click="item.editable && openEdit()">
          <div class="info-label">{{ item.label }} <span v-if="item.editable" style="font-size:10px;color:var(--color-primary)">✎</span></div>
          <div class="info-value">{{ item.value }}</div>
        </div>
      </a-col>
    </a-row>

    <a-row :gutter="16" class="animate-in" style="margin-top:16px" v-if="detail?.state === 'running' && vmStats">
      <a-col :span="12">
        <div class="info-card">
          <div class="info-label">CPU 使用率</div>
          <div style="display:flex;align-items:center;gap:12px;margin-top:8px">
            <a-progress :percent="(vmStats.cpu_usage || 0) / 100" :stroke-width="8" style="flex:1" />
            <span style="font-size:16px;font-weight:700;min-width:50px;text-align:right">{{ vmStats.cpu_usage }}%</span>
          </div>
        </div>
      </a-col>
      <a-col :span="12">
        <div class="info-card">
          <div class="info-label">内存使用</div>
          <div style="display:flex;align-items:center;gap:12px;margin-top:8px">
            <a-progress :percent="vmStats.mem_used && detail.memory ? vmStats.mem_used / detail.memory : 0" :stroke-width="8" color="#FF9500" style="flex:1" />
            <span style="font-size:16px;font-weight:700;min-width:80px;text-align:right">{{ vmStats.mem_used >= 1024 ? (vmStats.mem_used / 1024).toFixed(1) + ' GB' : vmStats.mem_used + ' MB' }}</span>
          </div>
        </div>
      </a-col>
    </a-row>

    <a-card class="animate-in" style="margin-top:16px" v-if="detail">
      <template #title>磁盘</template>
      <template #extra>
        <a-button size="small" type="primary" @click="showAttachDisk = true">挂载磁盘</a-button>
      </template>
      <a-table :data="detail.disks" row-key="target" :pagination="false" size="small">
        <template #columns>
          <a-table-column title="设备" data-index="target">
            <template #cell="{ record }"><code>{{ record.target }}</code></template>
          </a-table-column>
          <a-table-column title="类型">
            <template #cell="{ record }"><a-tag size="small">{{ record.device }}</a-tag></template>
          </a-table-column>
          <a-table-column title="源文件" data-index="source">
            <template #cell="{ record }"><code>{{ record.source || '(空)' }}</code></template>
          </a-table-column>
          <a-table-column title="总线">
            <template #cell="{ record }"><a-tag size="small" color="arcoblue">{{ record.bus }}</a-tag></template>
          </a-table-column>
          <a-table-column title="格式" data-index="format" />
          <a-table-column title="操作" :width="280">
            <template #cell="{ record }">
              <a-space>
                <a-button v-if="record.device === 'cdrom' && !record.source" size="small" @click="openAttachISO">挂载ISO</a-button>
                <a-button v-if="record.device === 'cdrom' && record.source" size="small" @click="openAttachISO">换碟</a-button>
                <a-button v-if="record.device === 'cdrom' && record.source" size="small" status="warning" @click="doDetachISO">弹出</a-button>
                <a-popconfirm v-if="record.device === 'disk' && record.target !== 'vda' && record.target !== 'sda'" content="确认卸载？" @ok="doDetachDisk(record.target)">
                  <a-button size="small" status="danger">卸载</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-card class="animate-in" style="margin-top:16px" v-if="detail">
      <template #title>网络接口</template>
      <template #extra>
        <a-button size="small" type="primary" @click="showAttachNIC = true">添加网卡</a-button>
      </template>
      <a-table :data="detail.nics" row-key="mac" :pagination="false" size="small">
        <template #columns>
          <a-table-column title="MAC 地址">
            <template #cell="{ record }"><code>{{ record.mac }}</code></template>
          </a-table-column>
          <a-table-column title="类型" data-index="type" />
          <a-table-column title="网络/网桥" data-index="source">
            <template #cell="{ record }"><span style="font-weight:500">{{ record.source }}</span></template>
          </a-table-column>
          <a-table-column title="模型">
            <template #cell="{ record }"><a-tag size="small" color="green">{{ record.model }}</a-tag></template>
          </a-table-column>
          <a-table-column title="操作" :width="100">
            <template #cell="{ record }">
              <a-popconfirm v-if="detail!.nics.length > 1" content="确认移除？" @ok="doDetachNIC(record.mac)">
                <a-button size="small" status="danger">移除</a-button>
              </a-popconfirm>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- Modals -->
    <a-modal v-model:visible="showAttachDisk" title="挂载磁盘" @ok="onAttachDisk" :ok-loading="attaching" unmount-on-close>
      <a-form :model="diskForm" layout="vertical">
        <a-form-item label="磁盘镜像路径" required><a-input v-model="diskForm.source" placeholder="/var/lib/libvirt/images/data.qcow2" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="目标设备"><a-input v-model="diskForm.target" placeholder="vdb" /></a-form-item></a-col>
          <a-col :span="12">
            <a-form-item label="总线">
              <a-select v-model="diskForm.bus">
                <a-option value="virtio">virtio</a-option>
                <a-option value="scsi">scsi</a-option>
                <a-option value="sata">sata</a-option>
                <a-option value="ide">ide</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showAttachISO" title="挂载 ISO" @ok="onAttachISO" :ok-loading="attaching" unmount-on-close>
      <a-form layout="vertical">
        <a-form-item label="选择 ISO">
          <a-select v-model="selectedISO" placeholder="选择 ISO 镜像">
            <a-option v-for="iso in isos" :key="iso.path" :value="iso.path">{{ iso.name }}</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showAttachNIC" title="添加网卡" @ok="onAttachNIC" :ok-loading="attaching" unmount-on-close>
      <a-form :model="nicForm" layout="vertical">
        <a-form-item label="网络模式">
          <a-select v-model="nicForm.mode">
            <a-option value="network">NAT (虚拟网络)</a-option>
            <a-option value="bridge">桥接</a-option>
            <a-option value="macvtap">macvtap</a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="nicForm.mode === 'network'" label="虚拟网络" required>
          <a-select v-model="nicForm.network" placeholder="选择网络">
            <a-option v-for="net in networks" :key="net.name" :value="net.name">{{ net.name }}</a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="nicForm.mode === 'bridge'" label="网桥" required>
          <a-select v-model="nicForm.bridge" placeholder="选择网桥">
            <a-option v-for="br in bridges" :key="br.name" :value="br.name">{{ br.name }} ({{ br.ip || 'no ip' }})</a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="nicForm.mode === 'macvtap'" label="物理网卡" required>
          <a-select v-model="nicForm.dev" placeholder="选择网卡">
            <a-option v-for="nic in hostNICs" :key="nic.name" :value="nic.name">{{ nic.name }} ({{ nic.ip || 'no ip' }})</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="网卡模型">
          <a-select v-model="nicForm.model">
            <a-option value="virtio">virtio</a-option>
            <a-option value="e1000">e1000</a-option>
            <a-option value="rtl8139">rtl8139</a-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showEdit" title="修改配置" @ok="onEdit" :ok-loading="editing" unmount-on-close>
      <a-alert v-if="detail?.state === 'running'" style="margin-bottom:16px">运行中修改将尝试热生效，部分场景需重启</a-alert>
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="CPU (核)">
          <a-input-number v-model="editForm.cpu" :min="1" :max="128" />
        </a-form-item>
        <a-form-item label="内存 (MB)">
          <a-input-number v-model="editForm.memory" :min="128" :max="1048576" :step="256" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showClone" title="克隆虚拟机" @ok="onClone" :ok-loading="cloning" unmount-on-close>
      <a-form :model="cloneForm" layout="vertical">
        <a-form-item label="新虚拟机名称" required>
          <a-input v-model="cloneForm.newName" :placeholder="vmName + '-clone'" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showRename" title="重命名虚拟机" @ok="onRename" :ok-loading="renaming" unmount-on-close>
      <a-form :model="renameForm" layout="vertical">
        <a-form-item label="新名称" required>
          <a-input v-model="renameForm.newName" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { vmApi, type VMDetail } from '../../api/vm'
import { isoApi, type ISOFile } from '../../api/iso'
import { networkApi, type Network } from '../../api/network'
import { hostApi } from '../../api/host'
import { bridgeApi, type Bridge } from '../../api/bridge'
import { errMsg } from '../../api/http'
import { Message } from '@arco-design/web-vue'
import { IconLeft } from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const vmName = computed(() => route.params.name as string)

const detail = ref<VMDetail | null>(null)
const vmStats = ref<{ cpu_usage: number; mem_used: number } | null>(null)
const isos = ref<ISOFile[]>([])
const networks = ref<Network[]>([])
const autostart = ref(false)
const showAttachDisk = ref(false)
const showAttachISO = ref(false)
const showAttachNIC = ref(false)
const attaching = ref(false)
const selectedISO = ref('')
const diskForm = reactive({ source: '', target: 'vdb', bus: 'virtio' })
const nicForm = reactive({ mode: 'network', network: '', bridge: '', dev: '', model: 'virtio' })
const hostNICs = ref<{ name: string; ip: string }[]>([])
const bridges = ref<Bridge[]>([])
const showEdit = ref(false)
const editing = ref(false)
const editForm = reactive({ cpu: 1, memory: 1024 })
const showClone = ref(false)
const cloning = ref(false)
const cloneForm = reactive({ newName: '' })
const showRename = ref(false)
const renaming = ref(false)
const renameForm = reactive({ newName: '' })

const stateText = (s: string) => ({ running: '运行中', shutoff: '已关机', paused: '已暂停' }[s] || s)

const infoCards = computed(() => {
  if (!detail.value) return []
  return [
    { label: 'CPU', value: detail.value.cpu + ' 核', editable: true },
    { label: '内存', value: detail.value.memory >= 1024 ? (detail.value.memory / 1024).toFixed(1) + ' GB' : detail.value.memory + ' MB', editable: true },
    { label: '架构', value: detail.value.arch, editable: false },
    { label: '启动设备', value: detail.value.boot, editable: false },
  ]
})

const loadDetail = async () => {
  try { detail.value = await vmApi.detail(vmName.value) } catch (e: any) { Message.error(errMsg(e, '加载失败')) }
  try { const vm = await vmApi.get(vmName.value); vmStats.value = { cpu_usage: vm.cpu_usage, mem_used: vm.mem_used } } catch {}
}
const hasISO = computed(() => detail.value?.disks?.some(d => d.device === 'cdrom' && d.source) ?? false)
const doFinishInstall = async () => { try { await vmApi.finishInstall(vmName.value); Message.success('已完成安装设置，下次启动将从硬盘引导'); loadDetail() } catch(e: any) { Message.error(errMsg(e, '操作失败')) } }

const pendingState = ref('')

const doAction = async (action: 'start' | 'shutdown' | 'destroy' | 'reboot' | 'suspend' | 'resume') => {
  const tips: Record<string, string> = { shutdown: '关机信号已发送', destroy: '已强制关机', reboot: '重启信号已发送' }
  const transient: Record<string, string> = { start: '启动中', shutdown: '关机中', destroy: '关机中', reboot: '重启中', suspend: '暂停中', resume: '恢复中' }
  try {
    await vmApi[action](vmName.value)
    Message.success(tips[action] || '操作成功')
    pendingState.value = transient[action] || '操作中'
    let tries = 0
    const poll = setInterval(async () => {
      tries++
      await loadDetail()
      const s = detail.value?.state
      const done = (action === 'shutdown' && s === 'shutoff') || (action === 'destroy' && s === 'shutoff') || (action === 'start' && s === 'running') || (action === 'reboot' && tries > 2) || (action === 'suspend' && s === 'paused') || (action === 'resume' && s === 'running')
      if (done || tries >= 20) { clearInterval(poll); pendingState.value = '' }
    }, 2000)
  } catch(e: any) { Message.error(errMsg(e, '操作失败')) }
}
const loadAutostart = async () => { try { const r = await vmApi.getAutostart(vmName.value); autostart.value = r.autostart } catch {} }
const onAutostartChange = async (v: boolean | string | number) => { try { await vmApi.setAutostart(vmName.value, v as boolean); Message.success(v ? '已开启自动启动' : '已关闭自动启动') } catch(e: any) { Message.error(errMsg(e, '设置失败')); autostart.value = !v } }
const openVNC = () => {
  const route = router.resolve({ name: 'vnc', params: { name: vmName.value } })
  window.open(route.href, '_blank')
}
const loadISOs = async () => { try { isos.value = await isoApi.list() } catch {} }
const loadNetworks = async () => { try { networks.value = await networkApi.list() } catch {} }
const loadHostNICs = async () => { try { hostNICs.value = await hostApi.nics() } catch {} }
const loadBridges = async () => { try { bridges.value = await bridgeApi.list() } catch {} }

const openEdit = () => {
  if (!detail.value) return
  editForm.cpu = detail.value.cpu
  editForm.memory = detail.value.memory
  showEdit.value = true
}
const onEdit = async () => {
  editing.value = true
  try { await vmApi.update(vmName.value, editForm); Message.success('修改成功'); showEdit.value = false; loadDetail() } catch(e: any) { Message.error(errMsg(e, '修改失败')) }
  editing.value = false
}

const openAttachISO = () => { loadISOs(); selectedISO.value = ''; showAttachISO.value = true }

const onAttachDisk = async () => {
  attaching.value = true
  try { await vmApi.attachDisk(vmName.value, diskForm); Message.success('挂载成功'); showAttachDisk.value = false; Object.assign(diskForm, { source: '', target: 'vdb', bus: 'virtio' }); loadDetail() } catch(e: any) { Message.error(errMsg(e, '挂载失败')) }
  attaching.value = false
}
const onAttachISO = async () => {
  if (!selectedISO.value) { Message.warning('请选择 ISO'); return }
  attaching.value = true
  try {
    await vmApi.attachISO(vmName.value, selectedISO.value)
    Message.success(detail.value?.state === 'running' ? '挂载成功，需重启生效' : '挂载成功')
    showAttachISO.value = false; loadDetail()
  } catch(e: any) { Message.error(errMsg(e, '挂载失败')) }
  attaching.value = false
}
const doDetachISO = async () => { try { await vmApi.detachISO(vmName.value); Message.success('已弹出'); loadDetail() } catch(e: any) { Message.error(errMsg(e, '操作失败')) } }
const doDetachDisk = async (target: string) => { try { await vmApi.detachDisk(vmName.value, target); Message.success('已卸载'); loadDetail() } catch(e: any) { Message.error(errMsg(e, '卸载失败')) } }
const onAttachNIC = async () => {
  attaching.value = true
  try { await vmApi.attachNIC(vmName.value, nicForm); Message.success('添加成功'); showAttachNIC.value = false; Object.assign(nicForm, { mode: 'network', network: '', bridge: '', dev: '', model: 'virtio' }); loadDetail() } catch(e: any) { Message.error(errMsg(e, '添加失败')) }
  attaching.value = false
}
const doDetachNIC = async (mac: string) => { try { await vmApi.detachNIC(vmName.value, mac); Message.success('已移除'); loadDetail() } catch(e: any) { Message.error(errMsg(e, '移除失败')) } }

const onClone = async () => {
  if (!cloneForm.newName.trim()) { Message.warning('请输入名称'); return }
  cloning.value = true
  const msgId = `clone-${Date.now()}`
  Message.loading({ content: `正在克隆，复制磁盘中...`, id: msgId, duration: 0 })
  try {
    await vmApi.clone(vmName.value, cloneForm.newName)
    Message.success({ content: '克隆成功', id: msgId }); showClone.value = false
  } catch(e: any) { Message.error({ content: errMsg(e, '克隆失败'), id: msgId }) }
  cloning.value = false
}

const onRename = async () => {
  if (!renameForm.newName.trim()) { Message.warning('请输入名称'); return }
  renaming.value = true
  try {
    await vmApi.rename(vmName.value, renameForm.newName)
    Message.success('重命名成功'); showRename.value = false
    router.replace({ name: 'vm-detail', params: { name: renameForm.newName } })
  } catch(e: any) { Message.error(errMsg(e, '重命名失败')) }
  renaming.value = false
}

watch(vmName, () => { loadDetail(); loadNetworks(); loadHostNICs(); loadBridges(); loadAutostart() })
onMounted(() => { loadDetail(); loadNetworks(); loadHostNICs(); loadBridges(); loadAutostart() })
</script>

<style scoped>
.detail-header {
  display: flex;
  align-items: center;
  margin-bottom: 20px;
}
.vm-name {
  font-size: 20px;
  font-weight: 700;
  color: #1d1d1f;
  margin: 0 10px 0 0;
  letter-spacing: -0.3px;
}
.header-info {
  display: flex;
  align-items: center;
  gap: 8px;
}
.info-card {
  background: #fff;
  border-radius: 12px;
  padding: 16px 20px;
  box-shadow: 0 1px 2px rgba(0,0,0,0.04);
}
.info-card.clickable {
  cursor: pointer;
  transition: background 0.15s;
}
.info-card.clickable:hover {
  background: #f5f5f7;
}
.info-label {
  font-size: 12px;
  font-weight: 600;
  color: #86868b;
  letter-spacing: -0.1px;
}
.info-value {
  font-size: 20px;
  font-weight: 700;
  color: #1d1d1f;
  margin-top: 4px;
  letter-spacing: -0.3px;
}
code {
  font-family: 'SF Mono', 'Fira Code', monospace;
  font-size: 12px;
  background: rgba(0,0,0,0.04);
  padding: 2px 6px;
  border-radius: 4px;
}
</style>
