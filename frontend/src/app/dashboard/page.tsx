import { useParams } from 'react-router-dom'
import { BalanceWidget } from '@/components/dashboard/BalanceWidget'
import { ProgressWidget } from '@/components/dashboard/ProgressWidget'
import { RatingWidget } from '@/components/dashboard/RatingWidget'
import { BingoWidget } from '@/components/dashboard/BingoWidget'
import { useBalance } from '@/hooks/useBalance'
import { useSubmissions } from '@/hooks/useSubmissions'

/**
 * Index child of /d/:seasonId — widget grid.
 * Season chip is rendered by DashboardLayout above this.
 */
export default function DashboardPage() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)

  const { data: balance, isLoading: balanceLoading } = useBalance(seasonId)
  const { data: submissions, isLoading: submissionsLoading } = useSubmissions(seasonId)

  const coins = balance?.balance ?? 0

  return (
    <>
      {/* Widget grid: 2 cols on sm+, 1 col on mobile */}
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <BalanceWidget coins={coins} isLoading={balanceLoading} seasonId={seasonId} />
        <RatingWidget isLoading={false} />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <ProgressWidget coins={coins} isLoading={balanceLoading} />
        <BingoWidget
          approvedCount={submissions?.approvedCount ?? 0}
          totalCount={submissions?.totalCount ?? 0}
          isLoading={submissionsLoading}
        />
      </div>
    </>
  )
}
