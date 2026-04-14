
import { useQuery } from '@tanstack/react-query'
import { balanceApi } from '@/lib/api/balance'

export function useMyBalances(enabled = true) {
  return useQuery({
    queryKey: ['balance', 'my'],
    queryFn: balanceApi.listMy,
    enabled,
  })
}
