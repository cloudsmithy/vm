import http from './http'

export interface DiskInfo {
  mount: string
  device: string
  total: number
  used: number
  available: number
  percent: number
}

export interface HostInfo {
  hostname: string
  cpu_model: string
  cpu_count: number
  cpu_usage: number
  memory_total: number
  memory_free: number
  vm_running: number
  vm_total: number
  uptime: number
  load_avg: [number, number, number]
  disks: DiskInfo[]
}

export const hostApi = {
  info: () => http.get<any, HostInfo>('/host/info'),
  nics: () => http.get<any, { name: string; mac: string; ip: string; up: boolean }[]>('/host/nics'),
}
