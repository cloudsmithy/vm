import axios from 'axios'

const http = axios.create({ baseURL: '/api', timeout: 300000 })

http.interceptors.response.use(
  (res) => res.data,
  (err) => {
    console.error(err)
    return Promise.reject(err)
  }
)

export default http
