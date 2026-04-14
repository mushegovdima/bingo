import { useMutation, useQueryClient } from '@tanstack/react-query'
import { seasonsApi, CreateSeasonRequest, UpdateSeasonRequest } from '@/lib/api/seasons'

export function useCreateSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateSeasonRequest) => seasonsApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seasons'] })
    },
  })
}

export function useUpdateSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateSeasonRequest }) =>
      seasonsApi.update(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seasons'] })
    },
  })
}

export function useDeleteSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (id: number) => seasonsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['seasons'] })
    },
  })
}
