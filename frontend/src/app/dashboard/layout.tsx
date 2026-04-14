import { Outlet, useParams } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'
import { SeasonPicker } from '@/components/dashboard/SeasonPicker'
import { useSeasonById } from '@/hooks/useSeasonById'
import s from './dashboard.module.scss'

/**
 * Shared layout for all /d/:seasonId/* routes.
 * Renders the season chip above each child route's content.
 */
export default function DashboardLayout() {
  const { seasonId: seasonIdStr } = useParams<{ seasonId: string }>()
  const seasonId = Number(seasonIdStr)
  const { data: season, isLoading } = useSeasonById(seasonId)

  return (
    <ProtectedPage>
      <AppLayout>
        <div className="flex flex-col gap-4">
          <div className="flex justify-center">
            {season && (
              <SeasonPicker currentSeasonId={seasonId} currentTitle={season.title} />
            )}
            {!isLoading && !season && (
              <p className={s.noSeason}>Сезон не найден</p>
            )}
          </div>
          <Outlet />
        </div>
      </AppLayout>
    </ProtectedPage>
  )
}
