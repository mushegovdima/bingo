import apiClient from './client'
import { TelegramAuthData, User } from '@/types'

export const authApi = {
  me: async (): Promise<User | null> => {
    try {
      const res = await apiClient.get<User>('/auth/me')
      return res.data
    } catch {
      return null
    }
  },

  login: async (data: TelegramAuthData): Promise<User> => {
    const res = await apiClient.post<User>('/auth/login', data)
    return res.data
  },

  logout: async (): Promise<void> => {
    await apiClient.post('/auth/logout')
  },
}
