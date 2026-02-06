import http from './http'

export interface StoragePool {
  name: string
  uuid: string
  active: boolean
  type: string
  path: string
  capacity: number
  allocation: number
  available: number
}

export const storageApi = {
  list: () => http.get<any, StoragePool[]>('/storage-pools'),
  create: (data: { name: string; path?: string }) =>
    http.post('/storage-pools', data),
  start: (name: string) => http.post(`/storage-pools/${name}/start`),
  stop: (name: string) => http.post(`/storage-pools/${name}/stop`),
  delete: (name: string) => http.delete(`/storage-pools/${name}`),
}
