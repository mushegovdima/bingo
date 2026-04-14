import { useAuth } from '@/hooks/useAuth'
import { Spinner } from '@heroui/react'
import { useNavigate } from 'react-router-dom'
import { useEffect } from 'react'

interface Props {
  children: React.ReactNode
  /** If true, the page is only accessible to managers. */
  requireManager?: boolean
}

/**
 * Guards a page: redirects to '/' when unauthenticated,
 * or shows an access-denied message when manager role is required.
 */
export function ProtectedPage({ children, requireManager = false }: Props) {
  const { user, isLoading, isAuthenticated, isManager } = useAuth()
  const navigate = useNavigate()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate('/login', { replace: true })
    }
  }, [isLoading, isAuthenticated, navigate])

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <Spinner size="lg" className="text-primary" />
      </div>
    )
  }

  if (!isAuthenticated) return null

  if (requireManager && !isManager) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh] gap-3">
        <span className="text-4xl">🔒</span>
        <p className="text-xl font-semibold">Доступ запрещён</p>
        <p className="text-default-400">Этот раздел доступен только для менеджеров.</p>
      </div>
    )
  }

  return <>{children}</>
}
