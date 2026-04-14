import apiClient from './client'
import { Task } from '@/types'

export interface CreateTaskRequest {
  season_id: number
  category: string
  title: string
  description: string
  reward_coins: number
  sort_order: number
  is_active: boolean
}

export interface UpdateTaskRequest {
  category?: string
  title?: string
  description?: string
  reward_coins?: number
  sort_order?: number
  is_active?: boolean
}

export const tasksApi = {
  listBySeason: async (seasonId: number): Promise<Task[]> => {
    const res = await apiClient.get<Task[]>('/tasks', {
      params: { season_id: seasonId },
    })
    return res.data
  },
  create: async (data: CreateTaskRequest): Promise<Task> => {
    const res = await apiClient.post<Task>('/tasks', data)
    return res.data
  },
  update: async (id: number, data: UpdateTaskRequest): Promise<Task> => {
    const res = await apiClient.patch<Task>(`/tasks/${id}`, data)
    return res.data
  },
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/tasks/${id}`)
  },
}
