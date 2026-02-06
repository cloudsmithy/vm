import http from './http'

export interface Snapshot {
  name: string
  description: string
  state: string
  created_at: number
  is_current: boolean
}

export const snapshotApi = {
  list: (vm: string) => http.get<any, Snapshot[]>(`/vms/${vm}/snapshots`),
  create: (vm: string, data: { name: string; description?: string }) =>
    http.post(`/vms/${vm}/snapshots`, data),
  delete: (vm: string, snap: string) => http.delete(`/vms/${vm}/snapshots/${snap}`),
  revert: (vm: string, snap: string) => http.post(`/vms/${vm}/snapshots/${snap}/revert`),
}
