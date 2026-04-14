import { useMutation, useQueryClient } from '@tanstack/react-query'
import { tasksApi, CreateTaskRequest, UpdateTaskRequest } from '@/lib/api/tasks'

export function useCreateTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateTaskRequest) => tasksApi.create(data),
    onSuccess: (_task, vars) => {
      queryClient.invalidateQueries({ queryKey: ['tasks', 'season', vars.season_id] })
    },
  })
}

export function useUpdateTask(seasonId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateTaskRequest }) =>
      tasksApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks', 'season', seasonId] })
    },
  })
}

export function useDeleteTask(seasonId: number) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => tasksApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['tasks', 'season', seasonId] })
    },
  })
}
