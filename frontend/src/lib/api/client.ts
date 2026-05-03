import axios from 'axios'

/**
 * Preconfigured axios instance.
 * - baseURL from NEXT_PUBLIC_API_URL
 * - withCredentials so the session cookie is sent on every request
 */
const apiClient = axios.create({
  // Relative /api is proxied by nginx to the Go backend (same origin → cookies work in all browsers).
  // VITE_API_URL can override this for prod deployments where nginx handles the proxy itself.
  baseURL: import.meta.env.VITE_API_URL || '/api/v1',
  withCredentials: true,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Normalize error messages from { error: "..." } responses
// and attach the HTTP status so callers can distinguish 401 from 5xx.
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    const message =
      error.response?.data?.error ?? error.message ?? 'Unexpected error'
    const err = new Error(message) as Error & { status?: number }
    err.status = error.response?.status
    return Promise.reject(err)
  },
)

export default apiClient
