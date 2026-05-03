import apiClient from './client'
import { TelegramAuthData, User } from '@/types'

export const authApi = {
  me: async (): Promise<User | null> => {
    try {
      const res = await apiClient.get<User>('/auth/me')
      return res.data
    } catch (e: unknown) {
      // 401 = not logged in, return null so useAuth treats user as guest.
      // Any other error (5xx, network) — rethrow so the query enters error
      // state without falsely clearing authentication.
      if ((e as { status?: number }).status === 401) return null
      throw e
    }
  },

  login: async (data: TelegramAuthData): Promise<User> => {
    const res = await apiClient.post<User>('/auth/login', data)
    return res.data
  },

  loginWebApp: async (initData: string): Promise<User> => {
    const res = await apiClient.post<User>('/auth/login/webapp', { init_data: initData })
    return res.data
  },

  logout: async (): Promise<void> => {
    await apiClient.post('/auth/logout')
  },
}
