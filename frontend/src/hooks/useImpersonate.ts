import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { usersApi } from '@/lib/api/users'

export function useImpersonate() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  return useMutation({
    mutationFn: (userId: number) => usersApi.impersonate(userId),
    onSuccess: () => {
      // Invalidate all cached data so it reloads under the new user context
      queryClient.clear()
      navigate('/')
    },
  })
}
