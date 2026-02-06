import http from './http'

export interface StorageVolume {
  name: string
  path: string
  type: string
  capacity: number
  allocation: number
}

export const volumeApi = {
  list: (pool: string) => http.get<any, StorageVolume[]>(`/storage-pools/${pool}/volumes`),
  create: (data: { pool: string; name: string; capacity: number; format?: string }) =>
    http.post('/storage-volumes', data),
  delete: (pool: string, vol: string) => http.delete(`/storage-pools/${pool}/volumes/${vol}`),
}
