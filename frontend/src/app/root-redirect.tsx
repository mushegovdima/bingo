import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Spinner } from '@heroui/react'
import { useAuth } from '@/hooks/useAuth'
import { useMyBalances } from '@/hooks/useMyBalances'
import { useSeasons } from '@/hooks/useSeasons'

/**
 * Smart root redirect:
 * - Not authenticated → /login
 * - Manager → /manager/d/:firstActiveSeasonId (or /manager/seasons if none)
 * - Has balances → /d/:latestSeasonId
 * - No balances → /seasons (join screen)
 */
export default function RootRedirect() {
  const navigate = useNavigate()
  const { isAuthenticated, isManager, isLoading: authLoading } = useAuth()
  const { data: balances, isLoading: balancesLoading } = useMyBalances(isAuthenticated && !isManager)
  const { data: seasons, isLoading: seasonsLoading } = useSeasons()

  useEffect(() => {
    if (authLoading) return
    if (!isAuthenticated) {
      navigate('/login', { replace: true })
      return
    }
    if (isManager) {
      if (seasonsLoading) return
      const first = seasons?.find((s) => s.is_active) ?? seasons?.[0]
      if (first) {
        navigate(`/manager/d/${first.id}`, { replace: true })
      } else {
        navigate('/manager/seasons', { replace: true })
      }
      return
    }
    if (balancesLoading) return
    if (balances && balances.length > 0) {
      navigate(`/d/${balances[0].season_id}`, { replace: true })
    } else {
      navigate('/seasons', { replace: true })
    }
  }, [isAuthenticated, isManager, authLoading, balances, balancesLoading, seasons, seasonsLoading, navigate])

  return (
    <div className="flex items-center justify-center min-h-screen">
      <Spinner size="lg" />
    </div>
  )
}
