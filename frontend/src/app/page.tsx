import { useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spinner } from '@heroui/react'
import { useAuth } from '@/hooks/useAuth'
import { TelegramLoginButton } from '@/components/auth/TelegramLoginButton'
import { TelegramAuthData } from '@/types'

export default function LoginPage() {
  const { isAuthenticated, isLoading, login } = useAuth()
  const navigate = useNavigate()
  const handledRef = useRef(false)

  const handleAuth = useCallback(async (data: TelegramAuthData) => {
    try {
      await login(data)
      navigate('/', { replace: true })
    } catch (e) {
      console.error('[auth] login failed', e)
    }
  }, [login, navigate])

  // Telegram redirect mode: after auth Telegram redirects to /login with query params:
  // ?id=...&first_name=...&auth_date=...&hash=...
  // (official docs: https://core.telegram.org/widgets/login)
  useEffect(() => {
    if (handledRef.current) return
    const p = new URLSearchParams(window.location.search)
    const hash = p.get('hash')
    const id = p.get('id')
    const auth_date = p.get('auth_date')
    if (!hash || !id || !auth_date) return
    handledRef.current = true
    // Clean URL so refresh doesn't re-trigger
    window.history.replaceState(null, '', '/login')
    handleAuth({
      id: Number(id),
      first_name: p.get('first_name') ?? '',
      last_name: p.get('last_name') ?? undefined,
      username: p.get('username') ?? undefined,
      photo_url: p.get('photo_url') ?? undefined,
      auth_date: Number(auth_date),
      hash,
    })
  }, [handleAuth])

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate('/', { replace: true })
    }
  }, [isLoading, isAuthenticated, navigate])

  if (isLoading || isAuthenticated) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" className="text-primary" />
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4 bg-(--color-bg)">
      <div className="flex flex-col items-center gap-8 w-full max-w-sm">
        <div className="text-center">
          <h1 className="text-5xl font-extrabold tracking-tight text-(--color-primary)">BINGO</h1>
          <p className="text-sm text-(--color-text-muted) mt-2">Bingo Gamification Platform</p>
        </div>
        <div className="bg-(--color-surface) border border-(--color-border) shadow-(--shadow-md) rounded-2xl p-8 w-full flex flex-col items-center gap-5">
          <p className="text-(--color-text-muted) text-center">Войдите через Telegram, чтобы продолжить</p>
          <TelegramLoginButton onAuth={handleAuth}  />
        </div>
      </div>
    </div>
  )
}
