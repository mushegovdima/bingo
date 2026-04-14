
import { useQuery } from '@tanstack/react-query'
import { seasonsApi } from '@/lib/api/seasons'

export function useSeasonById(id: number | undefined) {
  return useQuery({
    queryKey: ['seasons', id],
    queryFn: () => seasonsApi.getById(id!),
    enabled: !!id,
  })
}
