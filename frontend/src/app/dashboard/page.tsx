import { useParams } from 'react-router-dom'
import { BalanceWidget } from '@/components/dashboard/BalanceWidget'
import { ProgressWidget } from '@/components/dashboard/ProgressWidget'
import { RatingWidget } from '@/components/dashboard/RatingWidget'
import { TasksWidget } from '@/components/dashboard/TasksWidget'

export default function DashboardPage() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)

  return (
    <>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <BalanceWidget seasonId={seasonId} />
        <RatingWidget seasonId={seasonId} />
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Left: combined bingo counter + task list */}
        <TasksWidget seasonId={seasonId} />
        {/* Right: progress bar */}
        <ProgressWidget seasonId={seasonId} />
      </div>
    </>
  )
}
