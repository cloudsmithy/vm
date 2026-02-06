<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="快照">
      <template #extra>
        <a-space>
          <a-select v-model="selectedVM" placeholder="选择虚拟机" style="width:180px" @change="loadSnapshots" allow-clear>
            <a-option v-for="vm in vms" :key="vm.name" :value="vm.name">{{ vm.name }}</a-option>
          </a-select>
          <a-button type="primary" size="small" :disabled="!selectedVM" @click="showCreate = true">创建快照</a-button>
        </a-space>
      </template>
      <template v-if="selectedVM">
        <a-table :data="snapshots" :loading="loading" row-key="name" :pagination="false">
          <template #columns>
            <a-table-column title="名称" data-index="name">
              <template #cell="{ record }">
                <span style="font-weight:500">{{ record.name }}</span>
                <a-tag v-if="record.is_current" size="small" color="green" style="margin-left:8px">当前</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="描述" data-index="description">
              <template #cell="{ record }">
                <span style="color:var(--apple-gray)">{{ record.description || '-' }}</span>
              </template>
            </a-table-column>
            <a-table-column title="状态">
              <template #cell="{ record }">
                <a-tag size="small">{{ record.state }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="创建时间">
              <template #cell="{ record }">{{ new Date(record.created_at * 1000).toLocaleString() }}</template>
            </a-table-column>
            <a-table-column title="操作" :width="160">
              <template #cell="{ record }">
                <a-space>
                  <a-popconfirm content="确认恢复到此快照？" @ok="doRevert(record.name)">
                    <a-button size="small" type="primary">恢复</a-button>
                  </a-popconfirm>
                  <a-popconfirm content="确认删除？" @ok="doDelete(record.name)">
                    <a-button size="small" status="danger">删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </a-table-column>
          </template>
        </a-table>
        <a-empty v-if="!loading && snapshots.length === 0" description="暂无快照" style="padding:40px 0" />
      </template>
      <a-empty v-else style="padding:60px 0">
        <template #description>
          <span style="color:var(--apple-gray)">请先选择一台虚拟机</span>
        </template>
      </a-empty>
    </a-card>

    <a-modal v-model:visible="showCreate" title="创建快照" @ok="onCreate" :ok-loading="creating" unmount-on-close>
      <a-form :model="form" layout="vertical">
        <a-form-item label="名称" required><a-input v-model="form.name" placeholder="snap-01" /></a-form-item>
        <a-form-item label="描述"><a-input v-model="form.description" placeholder="可选描述" /></a-form-item>
      </a-form>
    </a-modal>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { vmApi, type VM } from '../../api/vm'
import { snapshotApi, type Snapshot } from '../../api/snapshot'
import { Message } from '@arco-design/web-vue'

const vms = ref<VM[]>([])
const selectedVM = ref('')
const snapshots = ref<Snapshot[]>([])
const loading = ref(false)
const showCreate = ref(false)
const creating = ref(false)
const form = reactive({ name: '', description: '' })

const loadVMs = async () => { try { vms.value = await vmApi.list() } catch {} }
const loadSnapshots = async () => {
  if (!selectedVM.value) { snapshots.value = []; return }
  loading.value = true
  try { snapshots.value = await snapshotApi.list(selectedVM.value) } catch { snapshots.value = [] }
  loading.value = false
}

const doRevert = async (snap: string) => {
  try { await snapshotApi.revert(selectedVM.value, snap); Message.success('已恢复'); loadSnapshots() } catch { Message.error('恢复失败') }
}
const doDelete = async (snap: string) => {
  try { await snapshotApi.delete(selectedVM.value, snap); Message.success('已删除'); loadSnapshots() } catch { Message.error('删除失败') }
}
const onCreate = async () => {
  creating.value = true
  try {
    await snapshotApi.create(selectedVM.value, form); Message.success('创建成功'); showCreate.value = false
    Object.assign(form, { name: '', description: '' }); loadSnapshots()
  } catch { Message.error('创建失败') }
  creating.value = false
}

onMounted(loadVMs)
</script>
