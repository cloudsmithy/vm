<template>
  <div class="vm-detail">
    <div class="detail-header animate-in">
      <a-button @click="router.push({ name: 'vm' })" style="margin-right:12px">
        <template #icon><icon-left /></template>
      </a-button>
      <div class="header-info">
        <h2 class="vm-name">{{ vmName }}</h2>
        <a-badge v-if="detail" :status="detail.state === 'running' ? 'success' : detail.state === 'paused' ? 'warning' : 'default'" :text="stateText(detail.state)" />
      </div>
      <div style="flex:1" />
      <a-space>
        <a-button v-if="detail?.state === 'running'" @click="router.push({ name: 'vnc', params: { name: vmName } })">控制台</a-button>
        <a-button @click="loadDetail">刷新</a-button>
      </a-space>
    </div>

    <a-row :gutter="16" class="animate-in" v-if="detail">
      <a-col :span="6" v-for="item in infoCards" :key="item.label">
        <div class="info-card">
          <div class="info-label">{{ item.label }}</div>
          <div class="info-value">{{ item.value }}</div>
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
          <a-table-column title="操作" :width="220">
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
        <a-form-item label="网络" required>
          <a-select v-model="nicForm.network" placeholder="选择网络">
            <a-option v-for="net in networks" :key="net.name" :value="net.name">{{ net.name }}</a-option>
          </a-select>
        </a-form-item>
        <a-form-item label="模型">
          <a-select v-model="nicForm.model">
            <a-option value="virtio">virtio</a-option>
            <a-option value="e1000">e1000</a-option>
            <a-option value="rtl8139">rtl8139</a-option>
          </a-select>
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
import { Message } from '@arco-design/web-vue'
import { IconLeft } from '@arco-design/web-vue/es/icon'

const route = useRoute()
const router = useRouter()
const vmName = computed(() => route.params.name as string)

const detail = ref<VMDetail | null>(null)
const isos = ref<ISOFile[]>([])
const networks = ref<Network[]>([])

const showAttachDisk = ref(false)
const showAttachISO = ref(false)
const showAttachNIC = ref(false)
const attaching = ref(false)
const selectedISO = ref('')
const diskForm = reactive({ source: '', target: 'vdb', bus: 'virtio' })
const nicForm = reactive({ network: '', model: 'virtio' })

const stateText = (s: string) => ({ running: '运行中', shutoff: '已关机', paused: '已暂停' }[s] || s)

const infoCards = computed(() => {
  if (!detail.value) return []
  return [
    { label: 'CPU', value: detail.value.cpu + ' 核' },
    { label: '内存', value: detail.value.memory >= 1024 ? (detail.value.memory / 1024).toFixed(1) + ' GB' : detail.value.memory + ' MB' },
    { label: '架构', value: detail.value.arch },
    { label: '启动设备', value: detail.value.boot },
  ]
})

const loadDetail = async () => { try { detail.value = await vmApi.detail(vmName.value) } catch { Message.error('加载失败') } }
const loadISOs = async () => { try { isos.value = await isoApi.list() } catch {} }
const loadNetworks = async () => { try { networks.value = await networkApi.list() } catch {} }

const openAttachISO = () => { loadISOs(); selectedISO.value = ''; showAttachISO.value = true }

const onAttachDisk = async () => {
  attaching.value = true
  try { await vmApi.attachDisk(vmName.value, diskForm); Message.success('挂载成功'); showAttachDisk.value = false; Object.assign(diskForm, { source: '', target: 'vdb', bus: 'virtio' }); loadDetail() } catch { Message.error('挂载失败') }
  attaching.value = false
}
const onAttachISO = async () => {
  if (!selectedISO.value) { Message.warning('请选择 ISO'); return }
  attaching.value = true
  try {
    await vmApi.attachISO(vmName.value, selectedISO.value)
    Message.success(detail.value?.state === 'running' ? '挂载成功，需重启生效' : '挂载成功')
    showAttachISO.value = false; loadDetail()
  } catch { Message.error('挂载失败') }
  attaching.value = false
}
const doDetachISO = async () => { try { await vmApi.detachISO(vmName.value); Message.success('已弹出'); loadDetail() } catch { Message.error('操作失败') } }
const doDetachDisk = async (target: string) => { try { await vmApi.detachDisk(vmName.value, target); Message.success('已卸载'); loadDetail() } catch { Message.error('卸载失败') } }
const onAttachNIC = async () => {
  attaching.value = true
  try { await vmApi.attachNIC(vmName.value, nicForm); Message.success('添加成功'); showAttachNIC.value = false; Object.assign(nicForm, { network: '', model: 'virtio' }); loadDetail() } catch { Message.error('添加失败') }
  attaching.value = false
}
const doDetachNIC = async (mac: string) => { try { await vmApi.detachNIC(vmName.value, mac); Message.success('已移除'); loadDetail() } catch { Message.error('移除失败') } }

watch(vmName, () => { loadDetail(); loadNetworks() })
onMounted(() => { loadDetail(); loadNetworks() })
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
  box-shadow: 0 1px 3px rgba(0,0,0,0.06);
}
.info-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--apple-gray);
  text-transform: uppercase;
  letter-spacing: 0.3px;
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
