'use client'

import { Badge, Button, Chip, Popover, Spinner } from '@heroui/react'
import { useNavigate } from 'react-router-dom'
import { useMyBalances } from '@/hooks/useMyBalances'
import { useActiveSeasons } from '@/hooks/useActiveSeasons'
import { useJoinSeason } from '@/hooks/useJoinSeason'
import type { Season } from '@/types'
import s from './SeasonPicker.module.scss'
import { Check, ChevronDown } from 'lucide-react'

interface Props {
  currentSeasonId: number
  currentTitle: string
}

export function SeasonPicker({ currentSeasonId, currentTitle }: Props) {
  const navigate = useNavigate()

  const { data: myBalances, isLoading: balancesLoading } = useMyBalances()
  const { data: activeSeasons, isLoading: seasonsLoading } = useActiveSeasons()
  const { mutate: join, isPending: isJoining } = useJoinSeason()

  const joinedIds = new Set(myBalances?.map((b) => b.season_id) ?? [])
  const allBalances = myBalances ?? []
  const unjoinedSeasons: Season[] = activeSeasons?.filter((c) => !joinedIds.has(c.id)) ?? []
  const hasNew = unjoinedSeasons.length > 0
  const isLoading = balancesLoading || seasonsLoading
  const showDropDown = !(allBalances.length === 0 && unjoinedSeasons.length === 0)

  const handleJoin = (seasonId: number) => {
    join(seasonId, {
      onSuccess: () => navigate(`/d/${seasonId}`),
    })
  }

  const triggerChip = (
    <Chip size="lg" variant="primary">
      <Chip.Label>
        <span className={s.seasonName}>{currentTitle}</span>
      </Chip.Label>
      { showDropDown && <ChevronDown size={16} className={s.seasonName}/> }
    </Chip>
  )

  return (
    <div className={s.root}>
      <Popover isOpen={isJoining ? true : undefined}>
        <Popover.Trigger>
          <Badge.Anchor>
            {triggerChip}
            {hasNew && (
              <Badge color="accent" size="sm" variant="primary">
                new
              </Badge>
            )}
          </Badge.Anchor>
        </Popover.Trigger>

        <Popover.Content className={s.content}>
          <Popover.Dialog>
            {isLoading ? (
              <div className={s.spinnerWrap}>
                <Spinner size="sm" />
              </div>
            ) : (
              <>
                {/* All joined seasons */}
                {allBalances.length > 0 && (
                  <div className={s.section}>
                    {allBalances.map((b) => {
                      const isCurrent = b.season_id === currentSeasonId
                      return (
                        <Button
                          key={b.id}
                          className={`${s.seasonRow}${isCurrent ? ` ${s.seasonRowCurrent}` : ''}`}
                          onClick={() => navigate(`/d/${b.season_id}`)}
                        >
                          <span className={s.rowTitle}>{b.season.title}</span>
                          <span className={s.rowRight}>
                            <span className={s.rowBalance}>{b.balance.toLocaleString('ru-RU')} KC</span>
                            <Check size={14} className={`${s.checkIcon}${isCurrent ? '' : ` ${s.checkIconHidden}`}`} />
                          </span>
                        </Button>
                      )
                    })}
                  </div>
                )}

                {/* Unjoined seasons as chips */}
                {unjoinedSeasons.length > 0 && (
                  <>
                    {allBalances.length > 0 && <div className={s.divider} />}
                    <p className={s.sectionLabel}>Доступные сезоны</p>
                    <div className={s.chipsRow}>
                      {unjoinedSeasons.map((c) => (
                        <div
                          key={c.id}
                          className={s.seasonRow}
                          onClick={isJoining ? undefined : () => handleJoin(c.id)}
                        >
                          <span className={s.rowTitle}>{c.title}</span>
                          <Button size="sm" variant='primary' className={s.joinBtn} isDisabled={isJoining} onClick={isJoining ? undefined : () => handleJoin(c.id)}>
                            Начать
                          </Button>
                        </div>
                      ))}
                    </div>
                  </>
                )}

                {allBalances.length === 0 && unjoinedSeasons.length === 0 && (
                  <p className={s.empty}>Нет других сезонов</p>
                )}
              </>
            )}
          </Popover.Dialog>
        </Popover.Content>
      </Popover>
    </div>
  )
}
