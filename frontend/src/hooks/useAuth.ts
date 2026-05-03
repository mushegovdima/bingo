

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi } from '@/lib/api/auth'
import { TelegramAuthData } from '@/types'
import { useLocation, useNavigate } from 'react-router-dom'
import { isTelegramMiniApp } from '@/lib/telegram'

export function useAuth() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const { data: user, isPending, isError } = useQuery({
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
      queryClient.clear()
      if (isTelegramMiniApp()) {
        window.Telegram!.WebApp!.close()
      } else {
        navigate('/login', { replace: true })
      }
    },
  })

  return {
    user: user ?? undefined,
    // When the me-query errored for a non-401 reason (e.g. 5xx / network),
    // keep isLoading=true so RootRedirect shows a spinner instead of
    // redirecting to /login.
    isLoading: isPending || isError,
    isAuthenticated: !!user,
    isManager: user?.roles?.includes('manager') ?? false,
    login: loginMutation.mutateAsync,
    logout: logoutMutation.mutate,
    isLoggingIn: loginMutation.isPending,
    loginError: loginMutation.error,
  }
}
