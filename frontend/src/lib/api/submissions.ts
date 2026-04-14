import apiClient from './client'
import { TaskSubmission } from '@/types'

export const submissionsApi = {
  getAll: async (): Promise<TaskSubmission[]> => {
    const res = await apiClient.get<TaskSubmission[]>('/submissions')
    return res.data
  },
}
