<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="网络">
      <template #extra>
        <a-button type="primary" size="small" @click="showCreate = true">创建网络</a-button>
      </template>
      <a-table :data="networks" :loading="loading" row-key="uuid" :pagination="false">
        <template #columns>
          <a-table-column title="名称" data-index="name">
            <template #cell="{ record }">
              <span style="font-weight:500">{{ record.name }}</span>
            </template>
          </a-table-column>
          <a-table-column title="状态">
            <template #cell="{ record }">
              <a-badge :status="record.active ? 'success' : 'default'" :text="record.active ? '活跃' : '未激活'" />
            </template>
          </a-table-column>
          <a-table-column title="模式" data-index="forward">
            <template #cell="{ record }">
              <a-tag size="small" :color="record.forward ? 'arcoblue' : 'gray'">{{ record.forward || 'isolated' }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="网桥" data-index="bridge">
            <template #cell="{ record }"><code>{{ record.bridge || '-' }}</code></template>
          </a-table-column>
          <a-table-column title="子网" data-index="subnet">
            <template #cell="{ record }"><code>{{ record.subnet || '-' }}</code></template>
          </a-table-column>
          <a-table-column title="操作" :width="200">
            <template #cell="{ record }">
              <a-space>
                <a-button v-if="!record.active" size="small" type="primary" @click="doAction(record.name, 'start')">启动</a-button>
                <a-button v-if="record.active" size="small" status="warning" @click="doAction(record.name, 'stop')">停止</a-button>
                <a-popconfirm content="确认删除？" @ok="doDelete(record.name)">
                  <a-button size="small" status="danger">删除</a-button>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
      <a-empty v-if="!loading && networks.length === 0" description="暂无网络" style="padding:40px 0" />
    </a-card>

    <a-modal v-model:visible="showCreate" title="创建网络" @ok="onCreate" :ok-loading="creating" unmount-on-close>
      <a-form :model="form" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model="form.name" placeholder="mynet" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="网关地址">
              <a-input v-model="form.subnet" placeholder="192.168.100.1" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="子网掩码">
              <a-input v-model="form.netmask" placeholder="255.255.255.0" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="DHCP 起始">
              <a-input v-model="form.dhcp_start" placeholder="192.168.100.100" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="DHCP 结束">
              <a-input v-model="form.dhcp_end" placeholder="192.168.100.200" />
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { networkApi, type Network } from '../../api/network'
import { Message } from '@arco-design/web-vue'

const networks = ref<Network[]>([])
const loading = ref(false)
const showCreate = ref(false)
const creating = ref(false)
const form = reactive({ name: '', subnet: '', netmask: '', dhcp_start: '', dhcp_end: '' })

const load = async () => {
  loading.value = true
  try { networks.value = await networkApi.list() } catch { /* */ }
  loading.value = false
}

const doAction = async (name: string, action: 'start' | 'stop') => {
  try { await networkApi[action](name); Message.success('操作成功'); load() } catch { Message.error('操作失败') }
}

const doDelete = async (name: string) => {
  try { await networkApi.delete(name); Message.success('已删除'); load() } catch { Message.error('删除失败') }
}

const onCreate = async () => {
  creating.value = true
  try {
    await networkApi.create(form); Message.success('创建成功'); showCreate.value = false
    Object.assign(form, { name: '', subnet: '', netmask: '', dhcp_start: '', dhcp_end: '' }); load()
  } catch { Message.error('创建失败') }
  creating.value = false
}

onMounted(load)
</script>

<style scoped>
code { font-family: 'SF Mono', 'Fira Code', monospace; font-size: 12px; background: rgba(0,0,0,0.04); padding: 2px 6px; border-radius: 4px; }
</style>
