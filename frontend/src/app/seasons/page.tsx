import { Button, Card, Spinner } from '@heroui/react'
import { useNavigate } from 'react-router-dom'
import { AppLayout } from '@/components/layout/AppLayout'
import { ProtectedPage } from '@/components/common/ProtectedPage'
import { useMyBalances } from '@/hooks/useMyBalances'
import { useActiveSeasons } from '@/hooks/useActiveSeasons'
import { useJoinSeason } from '@/hooks/useJoinSeason'
import s from './seasons.module.scss'

function SeasonsContent() {
  const navigate = useNavigate()
  const { data: myBalances, isLoading: balancesLoading } = useMyBalances()
  const { data: activeSeasons, isLoading: seasonsLoading } = useActiveSeasons()
  const { mutate: join, isPending: isJoining } = useJoinSeason()

  const isLoading = balancesLoading || seasonsLoading

  const joinedIds = new Set(myBalances?.map((b) => b.season_id) ?? [])
  const unjoinedSeasons = activeSeasons?.filter((c) => !joinedIds.has(c.id)) ?? []

  const handleJoin = (seasonId: number) => {
    join(seasonId, {
      onSuccess: () => navigate(`/d/${seasonId}`),
    })
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div className={s.page}>
      {/* My seasons */}
      {myBalances && myBalances.length > 0 && (
        <section>
          <p className={s.sectionTitle}>Мои сезоны</p>
          <div className={s.grid}>
            {myBalances.map((b) => (
              <Card
                key={b.id}
                className={s.card}
                role="button"
                tabIndex={0}
                onClick={() => navigate(`/d/${b.season_id}`)}
                onKeyDown={(e) => e.key === 'Enter' && navigate(`/d/${b.season_id}`)}
              >
                <Card.Header className={s.cardHeader}>
                  <Card.Title className={s.cardTitle}>{b.season.title}</Card.Title>
                </Card.Header>
                <Card.Content className={s.cardContent}>
                  <span className={s.cardBalance}>{b.balance.toLocaleString('ru-RU')} KC</span>
                </Card.Content>
                <Card.Footer className={s.cardFooter}>
                  <span className={s.cardMeta}>
                    {new Date(b.season.start_date).toLocaleDateString('ru-RU')}
                    {' — '}
                    {new Date(b.season.end_date).toLocaleDateString('ru-RU')}
                  </span>
                  <span className={s.cardAction}>Перейти →</span>
                </Card.Footer>
              </Card>
            ))}
          </div>
        </section>
      )}

      {/* Available seasons to join */}
      {unjoinedSeasons.length > 0 && (
        <section>
          <p className={s.sectionTitle}>Доступные сезоны</p>
          <div className={s.grid}>
            {unjoinedSeasons.map((c) => (
              <Card key={c.id} className={`${s.card} ${s.cardUnjoined}`}>
                <Card.Header className={s.cardHeader}>
                  <Card.Title className={s.cardTitle}>{c.title}</Card.Title>
                </Card.Header>
                <Card.Content className={s.cardContent}>
                  <span className={s.cardMeta}>
                    {new Date(c.start_date).toLocaleDateString('ru-RU')}
                    {' — '}
                    {new Date(c.end_date).toLocaleDateString('ru-RU')}
                  </span>
                </Card.Content>
                <Card.Footer className={s.cardFooter}>
                  <Button
                    size="sm"
                    variant="primary"
                    isDisabled={isJoining}
                    onPress={() => handleJoin(c.id)}
                  >
                    {isJoining ? '...' : 'Вступить'}
                  </Button>
                </Card.Footer>
              </Card>
            ))}
          </div>
        </section>
      )}

      {(!myBalances || myBalances.length === 0) && unjoinedSeasons.length === 0 && (
        <p className={s.empty}>Нет доступных кампаний. Обратитесь к менеджеру.</p>
      )}
    </div>
  )
}

export default function SeasonsPage() {
  return (
    <ProtectedPage>
      <AppLayout>
        <SeasonsContent />
      </AppLayout>
    </ProtectedPage>
  )
}
