<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="ISO 镜像">
      <template #extra>
        <a-upload :custom-request="onUpload" :show-file-list="false" accept=".iso">
          <template #upload-button>
            <a-button type="primary" size="small">上传 ISO</a-button>
          </template>
        </a-upload>
      </template>
      <div v-if="uploading" class="upload-bar">
        <div style="font-size:13px;font-weight:500;margin-bottom:8px">正在上传... {{ uploadPercent }}%</div>
        <a-progress :percent="uploadPercent / 100" :stroke-width="6" color="var(--apple-blue)" />
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
import { ref, onMounted } from 'vue'
import { isoApi, type ISOFile } from '../../api/iso'
import { Message } from '@arco-design/web-vue'

const isos = ref<ISOFile[]>([])
const loading = ref(false)
const uploading = ref(false)
const uploadPercent = ref(0)

const load = async () => { loading.value = true; try { isos.value = await isoApi.list() } catch {} loading.value = false }

const onUpload = async (option: any) => {
  uploading.value = true; uploadPercent.value = 0
  try { await isoApi.upload(option.fileItem.file!, (p) => { uploadPercent.value = p }); Message.success('上传成功'); load() } catch { Message.error('上传失败') }
  uploading.value = false
}

const doDelete = async (name: string) => {
  try { await isoApi.delete(name); Message.success('已删除'); load() } catch { Message.error('删除失败') }
}

onMounted(load)
</script>

<style scoped>
code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; background: rgba(0,0,0,0.04); padding: 2px 6px; border-radius: 4px; }
.upload-bar { background: rgba(0,122,255,0.04); border-radius: 10px; padding: 16px; margin-bottom: 16px; }
</style>
