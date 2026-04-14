
import { useQuery } from '@tanstack/react-query'
import { seasonsApi } from '@/lib/api/seasons'

export function useActiveSeasons() {
  return useQuery({
    queryKey: ['seasons', 'active-list'],
    queryFn: () => seasonsApi.list(true),
  })
}
