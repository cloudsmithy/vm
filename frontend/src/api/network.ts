import http from './http'

export interface Network {
  name: string
  uuid: string
  active: boolean
  forward: string
  bridge: string
  subnet: string
}

export const networkApi = {
  list: () => http.get<any, Network[]>('/networks'),
  create: (data: { name: string; bridge?: string; subnet?: string; netmask?: string; dhcp_start?: string; dhcp_end?: string }) =>
    http.post('/networks', data),
  start: (name: string) => http.post(`/networks/${name}/start`),
  stop: (name: string) => http.post(`/networks/${name}/stop`),
  delete: (name: string) => http.delete(`/networks/${name}`),
}
