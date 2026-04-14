

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/lib/api/auth'
import { TelegramAuthData } from '@/types'
import { useLocation, useNavigate } from 'react-router-dom'

export function useAuth() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const { data: user, isPending } = useQuery({
    queryKey: ['auth', 'me'],
    queryFn: authApi.me,
    retry: false,
  })

  const loginMutation = useMutation({
    mutationFn: (data: TelegramAuthData) => authApi.login(data),
    onSuccess: (user) => {
      queryClient.setQueryData(['auth', 'me'], user)
    },
  })

  const logoutMutation = useMutation({
    mutationFn: authApi.logout,
    onSuccess: () => {
      navigate('/login', { replace: true })
      queryClient.clear()
    },
  })

  return {
    user: user ?? undefined,
    isLoading: isPending,
    isAuthenticated: !!user,
    isManager: user?.roles?.includes('manager') ?? false,
    login: loginMutation.mutateAsync,
    logout: logoutMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    loginError: loginMutation.error,
  }
}
