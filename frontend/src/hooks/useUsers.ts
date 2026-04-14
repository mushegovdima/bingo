import { useQuery } from '@tanstack/react-query'
import { usersApi } from '@/lib/api/users'

export function useUsers() {
  return useQuery({
    queryKey: ['admin', 'users'],
    queryFn: usersApi.list,
  })
}
