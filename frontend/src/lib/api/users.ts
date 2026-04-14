import apiClient from './client'
import { User, UserRole } from '@/types'

export interface UpdateUserRequest {
  name: string
  roles: UserRole[]
  is_blocked: boolean
}

export const usersApi = {
  list: async (): Promise<User[]> => {
    const res = await apiClient.get<User[]>('/admin/users')
    return res.data
  },

  update: async (id: number, data: UpdateUserRequest): Promise<User> => {
    const res = await apiClient.patch<User>(`/admin/users/${id}`, data)
    return res.data
  },

  impersonate: async (id: number): Promise<User> => {
    const res = await apiClient.post<User>(`/auth/impersonate/${id}`)
    return res.data
  },
}
