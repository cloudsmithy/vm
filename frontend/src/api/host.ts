import http from './http'

export interface HostInfo {
  hostname: string
  cpu_model: string
  cpu_count: number
  memory_total: number
  memory_free: number
  vm_running: number
  vm_total: number
}

export const hostApi = {
  info: () => http.get<any, HostInfo>('/host/info'),
}
