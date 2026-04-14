
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'

export function useJoinSeason() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (seasonId: number) => balanceApi.joinSeason(seasonId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['balance', 'my'] })
    },
  })
}
