import axios, { AxiosInstance } from 'axios'

let apiInstance: AxiosInstance | null = null

export function useApi(): AxiosInstance {
  if (!apiInstance) {
    apiInstance = axios.create({
      baseURL: '/api',
      timeout: 10000,
    })

    // Request interceptor: attach Bearer token
    apiInstance.interceptors.request.use((config) => {
      const token = localStorage.getItem('smartpanel_token')
      if (token) {
        config.headers.Authorization = `Bearer ${token}`
      }
      return config
    })

    // Response interceptor: handle 401
    apiInstance.interceptors.response.use(
      (response) => response,
      (error) => {
        if (error.response && error.response.status === 401) {
          // If token expired, clear token
          const currentPath = window.location.pathname
          if (currentPath.startsWith('/admin')) {
            localStorage.removeItem('smartpanel_token')
            window.location.href = '/login'
          }
        }
        return Promise.reject(error)
      }
    )
  }

  return apiInstance
}
