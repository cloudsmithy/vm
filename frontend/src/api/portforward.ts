import http from './http'

export interface PortForward {
  id: string
  protocol: string
  host_port: number
  host_port_end?: number
  vm_ip: string
  vm_port: number
  comment: string
}

export const portForwardApi = {
  list: () => http.get('/port-forwards').then(r => r.data as PortForward[]),
  add: (data: Partial<PortForward>) => http.post('/port-forwards', data),
  delete: (id: string) => http.delete(`/port-forwards/${encodeURIComponent(id)}`),
}
