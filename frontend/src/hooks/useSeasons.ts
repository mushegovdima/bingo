import { useQuery } from '@tanstack/react-query'
import { seasonsApi } from '@/lib/api/seasons'

export function useSeasons() {
  return useQuery({
    queryKey: ['seasons', 'all'],
    queryFn: () => seasonsApi.list(),
  })
}
