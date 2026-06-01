import axios from 'axios'

const client = axios.create({
    baseURL: '/v1',
    timeout: 15000,
    headers: { 'Content-Type': 'application/json' },
})

// ── Request interceptor: inject auth token ─────────────────────────────────
client.interceptors.request.use((config) => {
    const token = localStorage.getItem('access_token')
    if (token) {
        config.headers.Authorization = `Bearer ${token}`
    }
    return config
})

// ── Response interceptor: handle 401 + refresh ─────────────────────────────
let isRefreshing = false
let failedQueue = []

const processQueue = (error, token = null) => {
    failedQueue.forEach(({ resolve, reject }) => {
        if (error) reject(error)
        else resolve(token)
    })
    failedQueue = []
}

client.interceptors.response.use(
    (response) => response,
    async (error) => {
        const originalRequest = error.config
        if (error.response?.status === 401 && !originalRequest._retry) {
            if (isRefreshing) {
                return new Promise((resolve, reject) => {
                    failedQueue.push({ resolve, reject })
                }).then((token) => {
                    originalRequest.headers.Authorization = `Bearer ${token}`
                    return client(originalRequest)
                })
            }

            originalRequest._retry = true
            isRefreshing = true

            try {
                const refreshToken = localStorage.getItem('refresh_token')
                if (!refreshToken) throw new Error('No refresh token')

                const { data } = await axios.post('/v1/auth/refresh', {
                    refresh_token: refreshToken,
                })

                localStorage.setItem('access_token', data.access_token)
                localStorage.setItem('refresh_token', data.refresh_token)

                processQueue(null, data.access_token)
                originalRequest.headers.Authorization = `Bearer ${data.access_token}`
                return client(originalRequest)
            } catch (refreshErr) {
                processQueue(refreshErr, null)
                localStorage.removeItem('access_token')
                localStorage.removeItem('refresh_token')
                window.location.href = '/login'
                return Promise.reject(refreshErr)
            } finally {
                isRefreshing = false
            }
        }
        return Promise.reject(error)
    },
)

export default client
