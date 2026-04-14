import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spinner } from '@heroui/react'
import { useAuth } from '@/hooks/useAuth'
import { useMyBalances } from '@/hooks/useMyBalances'

/**
 * Smart root redirect:
 * - Not authenticated → /login
 * - Has balances → /d/:latestSeasonId
 * - No balances → /seasons (join screen)
 */
export default function RootRedirect() {
  const navigate = useNavigate()
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const { data: balances, isLoading: balancesLoading } = useMyBalances(isAuthenticated)

  useEffect(() => {
    if (authLoading) return
    if (!isAuthenticated) {
      navigate('/login', { replace: true })
      return
    }
    if (balancesLoading) return
    if (balances && balances.length > 0) {
      navigate(`/d/${balances[0].season_id}`, { replace: true })
    } else {
      navigate('/seasons', { replace: true })
    }
  }, [isAuthenticated, authLoading, balances, balancesLoading, navigate])

  return (
    <div className="flex items-center justify-center min-h-screen">
      <Spinner size="lg" />
    </div>
  )
}
