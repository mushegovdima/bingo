import apiClient from './client'
import { Template, TemplateHistory } from '@/types'

export const templatesApi = {
  list: async (): Promise<Template[]> => {
    const res = await apiClient.get<Template[]>('/templates')
    return res.data
  },

  get: async (codename: string): Promise<Template> => {
    const res = await apiClient.get<Template>(`/templates/${codename}`)
    return res.data
  },

  updateBody: async (codename: string, body: string): Promise<Template> => {
    const res = await apiClient.patch<Template>(`/templates/${codename}`, { body })
    return res.data
  },

  listHistory: async (codename: string): Promise<TemplateHistory[]> => {
    const res = await apiClient.get<TemplateHistory[]>(`/templates/${codename}/history`)
    return res.data
  },
}
