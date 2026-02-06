<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="存储">
      <template #extra>
        <a-button type="primary" size="small" @click="showCreate = true">创建存储池</a-button>
      </template>
      <a-table :data="pools" :loading="loading" row-key="name" :pagination="false" :expandable="expandable">
        <template #columns>
          <a-table-column title="名称" data-index="name">
            <template #cell="{ record }"><span style="font-weight:500">{{ record.name }}</span></template>
          </a-table-column>
          <a-table-column title="状态">
            <template #cell="{ record }">
              <a-badge :status="record.active ? 'success' : 'default'" :text="record.active ? '活跃' : '未激活'" />
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }"><code>{{ record.path || '-' }}</code></template>
          </a-table-column>
          <a-table-column title="使用情况" :width="200">
            <template #cell="{ record }">
              <template v-if="record.active && record.capacity">
                <a-progress :percent="record.allocation / record.capacity" :stroke-width="6" style="width:120px" />
                <span style="font-size:11px;color:var(--apple-gray);margin-left:8px">{{ record.allocation }}/{{ record.capacity }} GB</span>
              </template>
              <span v-else style="color:var(--apple-gray)">-</span>
            </template>
          </a-table-column>
          <a-table-column title="操作" :width="220">
            <template #cell="{ record }">
              <a-space>
                <a-button v-if="!record.active" size="small" type="primary" @click="doAction(record.name, 'start')">启动</a-button>
                <a-button v-if="record.active" size="small" status="warning" @click="doAction(record.name, 'stop')">停止</a-button>
                <a-button v-if="record.active" size="small" @click="openCreateVol(record.name)">新建卷</a-button>
                <a-popconfirm content="确认删除？" @ok="doDelete(record.name)">
                  <a-button size="small" status="danger">删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
        <template #expand-row="{ record: poolRecord }">
          <div style="padding:4px 0">
            <a-table :data="volumes[poolRecord.name] || []" :loading="volLoading[poolRecord.name]" row-key="name" :pagination="false" size="small">
              <template #columns>
                <a-table-column title="卷名" data-index="name">
                  <template #cell="{ record: vol }"><span style="font-weight:500">{{ vol.name }}</span></template>
                </a-table-column>
                <a-table-column title="路径" data-index="path">
                  <template #cell="{ record: vol }"><code>{{ vol.path }}</code></template>
                </a-table-column>
                <a-table-column title="格式" data-index="type">
                  <template #cell="{ record: vol }"><a-tag size="small">{{ vol.type }}</a-tag></template>
                </a-table-column>
                <a-table-column title="容量">
                  <template #cell="{ record: vol }">{{ vol.capacity }} GB</template>
                </a-table-column>
                <a-table-column title="操作" :width="100">
                  <template #cell="{ record: vol }">
                    <a-popconfirm content="确认删除此卷？" @ok="doDeleteVol(poolRecord.name, vol.name)">
                      <a-button size="small" status="danger">删除</a-button>
                    </a-popconfirm>
                  </template>
                </a-table-column>
              </template>
            </a-table>
          </div>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="showCreate" title="创建存储池" @ok="onCreate" :ok-loading="creating" unmount-on-close>
      <a-form :model="form" layout="vertical">
        <a-form-item label="名称" required><a-input v-model="form.name" placeholder="mypool" /></a-form-item>
        <a-form-item label="路径"><a-input v-model="form.path" placeholder="/var/lib/libvirt/images/mypool" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="showCreateVol" title="创建存储卷" @ok="onCreateVol" :ok-loading="creatingVol" unmount-on-close>
      <a-form :model="volForm" layout="vertical">
        <a-form-item label="卷名" required><a-input v-model="volForm.name" placeholder="disk01.qcow2" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="容量 (GB)"><a-input-number v-model="volForm.capacity" :min="1" style="width:100%" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="格式">
              <a-select v-model="volForm.format">
                <a-option value="qcow2">qcow2</a-option>
                <a-option value="raw">raw</a-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { storageApi, type StoragePool } from '../../api/storage'
import { volumeApi, type StorageVolume } from '../../api/volume'
import { Message } from '@arco-design/web-vue'

const pools = ref<StoragePool[]>([])
const loading = ref(false)
const showCreate = ref(false)
const creating = ref(false)
const form = reactive({ name: '', path: '' })
const volumes = ref<Record<string, StorageVolume[]>>({})
const volLoading = ref<Record<string, boolean>>({})
const showCreateVol = ref(false)
const creatingVol = ref(false)
const volForm = reactive({ pool: '', name: '', capacity: 20, format: 'qcow2' })

const expandable = reactive({
  expandedRowKeys: [] as string[],
  onExpand: (key: string, expanded: boolean) => {
    if (expanded) { expandable.expandedRowKeys = [...expandable.expandedRowKeys, key]; loadVolumes(key) }
    else { expandable.expandedRowKeys = expandable.expandedRowKeys.filter(k => k !== key) }
  },
})

const load = async () => { loading.value = true; try { pools.value = await storageApi.list() } catch {} loading.value = false }
const loadVolumes = async (pool: string) => { volLoading.value[pool] = true; try { volumes.value[pool] = await volumeApi.list(pool) } catch { volumes.value[pool] = [] } volLoading.value[pool] = false }

const doAction = async (name: string, action: 'start' | 'stop') => {
  try { await storageApi[action](name); Message.success('操作成功'); load() } catch { Message.error('操作失败') }
}
const doDelete = async (name: string) => {
  try { await storageApi.delete(name); Message.success('已删除'); load() } catch { Message.error('删除失败') }
}
const onCreate = async () => {
  creating.value = true
  try { await storageApi.create(form); Message.success('创建成功'); showCreate.value = false; Object.assign(form, { name: '', path: '' }); load() } catch { Message.error('创建失败') }
  creating.value = false
}
const openCreateVol = (pool: string) => { volForm.pool = pool; volForm.name = ''; volForm.capacity = 20; volForm.format = 'qcow2'; showCreateVol.value = true }
const onCreateVol = async () => {
  creatingVol.value = true
  try { await volumeApi.create(volForm); Message.success('创建成功'); showCreateVol.value = false; loadVolumes(volForm.pool) } catch { Message.error('创建失败') }
  creatingVol.value = false
}
const doDeleteVol = async (pool: string, vol: string) => {
  try { await volumeApi.delete(pool, vol); Message.success('已删除'); loadVolumes(pool) } catch { Message.error('删除失败') }
}

onMounted(load)
</script>

<style scoped>
code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; background: rgba(0,0,0,0.04); padding: 2px 6px; border-radius: 4px; }
</style>
