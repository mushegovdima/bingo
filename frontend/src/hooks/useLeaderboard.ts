import { useQuery } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'

export function useLeaderboard(seasonId: number | undefined) {
  return useQuery({
    queryKey: ['leaderboard', seasonId],
    queryFn: () => balanceApi.getLeaderboard(seasonId!),
    enabled: !!seasonId,
  })
}
