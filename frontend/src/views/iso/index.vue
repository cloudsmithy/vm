<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="ISO 镜像">
      <template #extra>
        <a-upload :custom-request="onUpload" :show-file-list="false" accept=".iso" :multiple="true" draggable>
          <template #upload-button>
            <a-button type="primary" size="small">上传 ISO</a-button>
          </template>
        </a-upload>
      </template>
      <div v-for="task in uploadTasks" :key="task.id" class="upload-bar">
        <div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:8px">
          <span style="font-size:13px;font-weight:500">{{ task.name }} — {{ task.percent }}%</span>
          <a-button size="mini" status="danger" @click="cancelUpload(task.id)">取消</a-button>
        </div>
        <a-progress :percent="task.percent / 100" :stroke-width="6" color="#165DFF" />
      </div>
      <a-table :data="isos" :loading="loading" row-key="name" :pagination="false">
        <template #columns>
          <a-table-column title="文件名" data-index="name">
            <template #cell="{ record }">
              <span style="font-weight:500">{{ record.name }}</span>
            </template>
          </a-table-column>
          <a-table-column title="路径" data-index="path">
            <template #cell="{ record }"><code>{{ record.path }}</code></template>
          </a-table-column>
          <a-table-column title="大小">
            <template #cell="{ record }">{{ (record.size / 1024 / 1024 / 1024).toFixed(2) }} GB</template>
          </a-table-column>
          <a-table-column title="操作" :width="100">
            <template #cell="{ record }">
              <a-popconfirm content="确认删除？" @ok="doDelete(record.name)">
                <a-button size="small" status="danger">删除</a-button>
              </a-popconfirm>
            </template>
          </a-table-column>
        </template>
      </a-table>
      <a-empty v-if="!loading && isos.length === 0" description="暂无 ISO 镜像" style="padding:40px 0" />
    </a-card>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { isoApi, type ISOFile } from '../../api/iso'
import { Message } from '@arco-design/web-vue'
import type { CancelTokenSource } from 'axios'

interface UploadTask { id: number; name: string; percent: number; cancel: CancelTokenSource }

const isos = ref<ISOFile[]>([])
const loading = ref(false)

let taskId = 0
const uploadTasks = ref<UploadTask[]>([])

const load = async () => { loading.value = true; try { isos.value = await isoApi.list() } catch {} loading.value = false }

const onUpload = async (option: any) => {
  const file = option.fileItem.file!
  const { source, promise } = isoApi.upload(file, (p) => { task.percent = p })
  const task = reactive<UploadTask>({ id: ++taskId, name: file.name, percent: 0, cancel: source })
  uploadTasks.value.push(task)
  try {
    await promise
    Message.success(`${file.name} 上传成功`); load()
  } catch (e: any) {
    if (e?.message !== 'canceled') Message.error(`${file.name} 上传失败`)
  }
  uploadTasks.value = uploadTasks.value.filter(t => t.id !== task.id)
}

const cancelUpload = (id: number) => {
  const task = uploadTasks.value.find(t => t.id === id)
  if (task) { task.cancel.cancel(); Message.info(`已取消 ${task.name}`) }
}

const doDelete = async (name: string) => {
  try { await isoApi.delete(name); Message.success('已删除'); load() } catch { Message.error('删除失败') }
}

onMounted(load)
</script>

<style scoped>
code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; background: rgba(0,0,0,0.04); padding: 2px 6px; border-radius: 4px; }
.upload-bar { background: #f5f5f7; border-radius: 12px; padding: 16px; margin-bottom: 16px; }
</style>
