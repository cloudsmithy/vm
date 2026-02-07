<template>
  <a-space direction="vertical" fill size="medium">
    <a-card title="端口转发">
      <template #extra>
        <a-button type="primary" size="small" @click="showAdd = true">添加规则</a-button>
      </template>
      <a-table :data="rules" :loading="loading" row-key="id" :pagination="false">
        <template #columns>
          <a-table-column title="协议" :width="80">
            <template #cell="{ record }"><a-tag size="small">{{ record.protocol.toUpperCase() }}</a-tag></template>
          </a-table-column>
          <a-table-column title="宿主机端口" :width="140">
            <template #cell="{ record }">{{ record.host_port_end ? `${record.host_port}-${record.host_port_end}` : record.host_port }}</template>
          </a-table-column>
          <a-table-column title="目标 VM IP" data-index="vm_ip" :width="160" />
          <a-table-column title="目标端口" :width="140">
            <template #cell="{ record }">
              <template v-if="record.host_port_end">{{ record.vm_port }}-{{ record.vm_port + (record.host_port_end - record.host_port) }}</template>
              <template v-else>{{ record.vm_port }}</template>
            </template>
          </a-table-column>
          <a-table-column title="备注" data-index="comment" />
          <a-table-column title="操作" :width="80">
            <template #cell="{ record }">
              <a-popconfirm content="确认删除此规则？" @ok="doDelete(record.id)">
                <a-button size="small" status="danger">删除</a-button>
              </a-popconfirm>
            </template>
          </a-table-column>
        </template>
      </a-table>
      <a-empty v-if="!loading && rules.length === 0" description="暂无端口转发规则" style="padding:40px 0" />
    </a-card>

    <a-modal v-model:visible="showAdd" title="添加端口转发" @ok="onAdd" :ok-loading="adding" unmount-on-close>
      <a-form :model="form" layout="vertical">
        <a-form-item label="模式">
          <a-radio-group v-model="form.mode">
            <a-radio value="single">单端口</a-radio>
            <a-radio value="range">端口范围</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="协议">
          <a-radio-group v-model="form.protocol">
            <a-radio value="tcp">TCP</a-radio>
            <a-radio value="udp">UDP</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="form.mode === 'single'" label="宿主机端口" required>
          <a-input-number v-model="form.host_port" :min="1" :max="65535" style="width:100%" />
        </a-form-item>
        <a-form-item v-else label="宿主机端口范围" required>
          <a-space>
            <a-input-number v-model="form.host_port" :min="1" :max="65535" placeholder="起始" style="width:120px" />
            <span>-</span>
            <a-input-number v-model="form.host_port_end" :min="1" :max="65535" placeholder="结束" style="width:120px" />
          </a-space>
        </a-form-item>
        <a-form-item label="目标虚拟机" required>
          <a-select v-model="form.vm_ip" placeholder="选择虚拟机" allow-search>
            <a-option v-for="vm in vmOptions" :key="vm.ip" :value="vm.ip">{{ vm.name }} ({{ vm.ip }})</a-option>
          </a-select>
        </a-form-item>
        <a-form-item v-if="form.mode === 'single'" label="目标端口" required>
          <a-input-number v-model="form.vm_port" :min="1" :max="65535" style="width:100%" />
        </a-form-item>
        <a-form-item v-else label="目标端口范围" required>
          <a-space>
            <a-input-number v-model="form.vm_port" :min="1" :max="65535" placeholder="起始" style="width:120px" />
            <span>-</span>
            <a-input-number :model-value="form.vm_port && form.host_port && form.host_port_end ? form.vm_port + (form.host_port_end - form.host_port) : undefined" disabled placeholder="自动计算" style="width:120px" />
          </a-space>
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model="form.comment" placeholder="可选" />
        </a-form-item>
      </a-form>
    </a-modal>
  </a-space>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { portForwardApi, type PortForward } from '../../api/portforward'
import { Message } from '@arco-design/web-vue'
import { errMsg } from '../../api/http'

const rules = ref<PortForward[]>([])
const loading = ref(false)
const showAdd = ref(false)
const adding = ref(false)
const vmOptions = ref<{ name: string; ip: string }[]>([])
const form = reactive({
  mode: 'single' as 'single' | 'range',
  protocol: 'tcp',
  host_port: undefined as number | undefined,
  host_port_end: undefined as number | undefined,
  vm_ip: '',
  vm_port: undefined as number | undefined,
  comment: '',
})

const load = async () => { loading.value = true; try { rules.value = await portForwardApi.list() || [] } catch {} loading.value = false }

const loadVMs = async () => {
  try {
    const res = await fetch('/api/networks/default/leases')
    if (res.ok) {
      const leases = await res.json()
      vmOptions.value = leases.map((l: any) => ({ name: l.hostname || l.mac, ip: l.ip }))
    }
  } catch {}
}

const onAdd = async () => {
  if (!form.host_port || !form.vm_ip || !form.vm_port) { Message.warning('请填写完整'); return }
  if (form.mode === 'range' && (!form.host_port_end || form.host_port_end <= form.host_port)) {
    Message.warning('结束端口必须大于起始端口'); return
  }
  adding.value = true
  try {
    const data: any = { protocol: form.protocol, host_port: form.host_port, vm_ip: form.vm_ip, vm_port: form.vm_port, comment: form.comment }
    if (form.mode === 'range') data.host_port_end = form.host_port_end
    await portForwardApi.add(data)
    Message.success('添加成功'); showAdd.value = false
    Object.assign(form, { mode: 'single', protocol: 'tcp', host_port: undefined, host_port_end: undefined, vm_ip: '', vm_port: undefined, comment: '' })
    load()
  } catch (e: any) { Message.error(errMsg(e, '添加失败')) }
  adding.value = false
}

const doDelete = async (id: string) => {
  try { await portForwardApi.delete(id); Message.success('已删除'); load() } catch (e: any) { Message.error(errMsg(e, '删除失败')) }
}

onMounted(() => { load(); loadVMs() })
</script>
