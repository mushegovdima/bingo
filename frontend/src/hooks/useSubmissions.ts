

import { useQuery } from '@tanstack/react-query'
import { submissionsApi } from '@/lib/api/submissions'

export function useSubmissions(seasonId: number | undefined) {
  return useQuery({
    queryKey: ['submissions', seasonId],
    queryFn: () => submissionsApi.getAll(),
    enabled: !!seasonId,
    select: (submissions) => {
      const forSeason = submissions.filter(
        (s) => s.season_id === seasonId,
      )
      return {
        all: forSeason,
        approvedCount: forSeason.filter((s) => s.status === 'approved').length,
        totalCount: forSeason.length,
      }
    },
  })
}
