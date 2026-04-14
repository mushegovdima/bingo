import { useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spinner } from '@heroui/react'
import { useAuth } from '@/hooks/useAuth'
import { TelegramLoginButton } from '@/components/auth/TelegramLoginButton'
import { TelegramAuthData } from '@/types'
import s from './login.module.scss'

/**
 * Login page (/login):
 * - If already authenticated → redirect to /
 * - If not → show Telegram Login Widget (callback mode)
 */
export default function LoginPage() {
  const { isAuthenticated, isLoading, login } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate('/', { replace: true })
    }
  }, [isLoading, isAuthenticated, navigate])

  const handleAuth = useCallback(async (data: TelegramAuthData) => {
    console.debug('LoginPage: received auth data from Telegram', data)
    await login(data)
    navigate('/')
  }, [login, navigate])

  if (isLoading || isAuthenticated) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" className="text-primary" />
      </div>
    )
  }

  return (
    <div className={s.page}>
      <div className="flex flex-col items-center gap-8 w-full max-w-sm">
        <div className="text-center">
          <h1 className={s.title}>BINGO</h1>
          <p className={s.subtitle}>Bingo Gamification Platform</p>
        </div>
        <div className={s.card}>
          <p className={s.hint}>Войдите через Telegram, чтобы продолжить</p>
          <TelegramLoginButton onAuth={handleAuth} />
        </div>
      </div>
    </div>
  )
}
