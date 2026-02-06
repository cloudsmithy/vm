import http from './http'
import axios from 'axios'

export interface ISOFile {
  name: string
  path: string
  size: number
}

export const isoApi = {
  list: () => http.get<any, ISOFile[]>('/isos'),
  upload: (file: File, onProgress?: (percent: number) => void) => {
    const form = new FormData()
    form.append('file', file)
    return axios.post('/api/isos/upload', form, {
      baseURL: '',
      timeout: 0,
      onUploadProgress: (e) => {
        if (onProgress && e.total) onProgress(Math.round((e.loaded / e.total) * 100))
      },
    })
  },
  delete: (name: string) => http.delete(`/isos/${name}`),
}
