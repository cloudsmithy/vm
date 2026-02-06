import http from './http'

export interface VM {
  name: string
  uuid: string
  state: string
  cpu: number
  memory: number
}

export interface VMDetail {
  name: string
  uuid: string
  state: string
  cpu: number
  memory: number
  disks: VMDisk[]
  nics: VMNIC[]
  boot: string
  arch: string
}

export interface VMDisk {
  device: string
  source: string
  target: string
  bus: string
  format: string
}

export interface VMNIC {
  type: string
  source: string
  mac: string
  model: string
}

export const vmApi = {
  list: () => http.get<any, VM[]>('/vms'),
  get: (name: string) => http.get<any, VM>(`/vms/${name}`),
  detail: (name: string) => http.get<any, VMDetail>(`/vms/${name}/detail`),
  start: (name: string) => http.post(`/vms/${name}/start`),
  shutdown: (name: string) => http.post(`/vms/${name}/shutdown`),
  destroy: (name: string) => http.post(`/vms/${name}/destroy`),
  reboot: (name: string) => http.post(`/vms/${name}/reboot`),
  suspend: (name: string) => http.post(`/vms/${name}/suspend`),
  resume: (name: string) => http.post(`/vms/${name}/resume`),
  delete: (name: string) => http.delete(`/vms/${name}`),
  create: (data: { name: string; cpu: number; memory: number; disk: number; os_type?: string; disk_bus?: string; net_model?: string; machine?: string; cpu_model?: string; clock?: string; virtio_iso?: string }) =>
    http.post('/vms', data),
  update: (name: string, data: { cpu?: number; memory?: number }) =>
    http.put(`/vms/${name}`, data),
  clone: (name: string, newName: string) =>
    http.post(`/vms/${name}/clone`, { new_name: newName }),
  attachDisk: (name: string, data: { source: string; target?: string; bus?: string }) =>
    http.post(`/vms/${name}/disks`, data),
  detachDisk: (name: string, target: string) =>
    http.delete(`/vms/${name}/disks/${target}`),
  attachNIC: (name: string, data: { network: string; model?: string }) =>
    http.post(`/vms/${name}/nics`, data),
  detachNIC: (name: string, mac: string) =>
    http.delete(`/vms/${name}/nics/${encodeURIComponent(mac)}`),
  attachISO: (name: string, path: string) =>
    http.post(`/vms/${name}/iso`, { path }),
  detachISO: (name: string) =>
    http.delete(`/vms/${name}/iso`),
}
