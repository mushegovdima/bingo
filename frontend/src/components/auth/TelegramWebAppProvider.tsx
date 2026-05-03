
import { useEffect, useRef, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { Spinner } from '@heroui/react'
import { authApi } from '@/lib/api/auth'
import { getTelegramWebApp, isTelegramMiniApp } from '@/lib/telegram'

/**
 * Initialises Telegram Mini App context.
 * - Calls WebApp.ready() and expand() so the app fills the Telegram window.
 * - If initData is present and the user is not yet authenticated, silently
 *   logs in via POST /auth/login/webapp so they never see the login screen.
 *
 * While the login attempt is in flight we render a spinner instead of
 * children — this prevents RootRedirect from seeing isAuthenticated=false
 * and bouncing the user to /login before the session is established.
 */
export function TelegramWebAppProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient()
  // If not in Mini App context we are immediately ready.
  const [ready, setReady] = useState(!isTelegramMiniApp())
  const attempted = useRef(false)

  useEffect(() => {
    const twa = getTelegramWebApp()
    if (!twa) return

    twa.ready()
    twa.expand()

    if (attempted.current) return
    attempted.current = true

    // Only auto-login if we don't already have an authenticated user cached
    const cached = queryClient.getQueryData(['auth', 'me'])
    if (cached) {
      setReady(true)
      return
    }

    authApi
      .loginWebApp(twa.initData)
      .then((user) => {
        queryClient.setQueryData(['auth', 'me'], user)
      })
      .catch(() => {
        // initData invalid or expired — fall through to normal login screen
      })
      .finally(() => {
        setReady(true)
      })
  }, [queryClient])

  if (!ready) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <Spinner size="lg" />
      </div>
    )
  }

  return <>{children}</>
}
