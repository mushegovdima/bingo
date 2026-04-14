

import { useQuery } from '@tanstack/react-query'
import { seasonsApi } from '@/lib/api/seasons'

export function useSeason() {
  return useQuery({
    queryKey: ['seasons', 'active'],
    queryFn: seasonsApi.getActive,
    throwOnError: false,
    retry: false,
  })
}
