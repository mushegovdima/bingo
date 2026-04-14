import axios from 'axios'

/**
 * Preconfigured axios instance.
 * - baseURL from NEXT_PUBLIC_API_URL
 * - withCredentials so the session cookie is sent on every request
 */
const apiClient = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:4010',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Normalize error messages from { error: "..." } responses
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const message =
      error.response?.data?.error ?? error.message ?? 'Unexpected error'
    return Promise.reject(new Error(message))
  },
)

export default apiClient
