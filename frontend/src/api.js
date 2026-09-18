import axios from 'axios'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => {
    const body = response.data
    if (body && typeof body === 'object' && 'success' in body) {
      if (body.success === false) {
        return Promise.reject(body)
      }
      return body.data
    }
    return body
  },
  (error) => {
    const status = error.response?.status
    const isAuthPath =
      typeof window !== 'undefined' &&
      (window.location.pathname === '/login' || window.location.pathname === '/signup')

    if (status === 401 && !isAuthPath) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (!isAuthPath && window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }

    const message =
      error.response?.data?.message ||
      error.message ||
      'Something went wrong. Please try again.'
    return Promise.reject(new Error(message))
  }
)

export default api