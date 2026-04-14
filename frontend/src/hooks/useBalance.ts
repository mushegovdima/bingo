

import { useQuery } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'

export function useBalance(seasonId: number | undefined) {
  return useQuery({
    queryKey: ['balance', seasonId],
    queryFn: () => balanceApi.getBalance(seasonId!),
    enabled: !!seasonId,
  })
}
