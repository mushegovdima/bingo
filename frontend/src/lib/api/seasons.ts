import apiClient from './client'
import { Season } from '@/types'

export interface CreateSeasonRequest {
  title: string
  start_date: string
  end_date: string
  is_active: boolean
}

export interface UpdateSeasonRequest {
  title?: string
  start_date?: string
  end_date?: string
  is_active?: boolean
}

export const seasonsApi = {
  getActive: async (): Promise<Season> => {
    const res = await apiClient.get<Season>('/seasons/active')
    return res.data
  },
  getById: async (id: number): Promise<Season> => {
    const res = await apiClient.get<Season>(`/seasons/${id}`)
    return res.data
  },
  list: async (activeOnly = false): Promise<Season[]> => {
    const res = await apiClient.get<Season[]>('/seasons', {
      params: activeOnly ? { active: 'true' } : undefined,
    })
    return res.data
  },
  create: async (data: CreateSeasonRequest): Promise<Season> => {
    const res = await apiClient.post<Season>('/seasons', data)
    return res.data
  },
  update: async (id: number, data: UpdateSeasonRequest): Promise<Season> => {
    const res = await apiClient.patch<Season>(`/seasons/${id}`, data)
    return res.data
  },
  delete: async (id: number): Promise<void> => {
    await apiClient.delete(`/seasons/${id}`)
  },
}
