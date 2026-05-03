import apiClient from './client'
import { TaskSubmission } from '@/types'

export interface SubmitTaskRequest {
  task_id: number
  comment: string
}

export const submissionsApi = {
  getAll: async (): Promise<TaskSubmission[]> => {
    const res = await apiClient.get<TaskSubmission[]>('/submissions')
    return res.data
  },
  getAllAdmin: async (): Promise<TaskSubmission[]> => {
    const res = await apiClient.get<TaskSubmission[]>('/submissions/all')
    return res.data
  },
  submit: async (data: SubmitTaskRequest): Promise<TaskSubmission> => {
    const res = await apiClient.post<TaskSubmission>('/submissions/submit', data)
    return res.data
  },
  approve: async (id: number): Promise<TaskSubmission> => {
    const res = await apiClient.post<TaskSubmission>(`/submissions/${id}/approve`)
    return res.data
  },
  reject: async (id: number, comment: string): Promise<TaskSubmission> => {
    const res = await apiClient.post<TaskSubmission>(`/submissions/${id}/reject`, { comment })
    return res.data
  },
}
