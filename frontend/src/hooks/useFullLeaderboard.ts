import { useQuery } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'

export function useFullLeaderboard(seasonId: number | undefined) {
  return useQuery({
    queryKey: ['leaderboard', 'full', seasonId],
    queryFn: () => balanceApi.getFullLeaderboard(seasonId!),
    enabled: !!seasonId,
  })
}
