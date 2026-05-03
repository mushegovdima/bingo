'use client'

import { Badge, Button, Chip, Popover, Spinner } from '@heroui/react'
import { useNavigate } from 'react-router-dom'
import { useMyBalances } from '@/hooks/useMyBalances'
import { useActiveSeasons } from '@/hooks/useActiveSeasons'
import { useJoinSeason } from '@/hooks/useJoinSeason'
import type { Season } from '@/types'
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
        <span className="text-(--color-primary) font-semibold">{currentTitle}</span>
      </Chip.Label>
      { showDropDown && <ChevronDown size={16} className="text-(--color-primary) font-semibold"/> }
    </Chip>
  )

  return (
    <div className="flex justify-center">
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

        <Popover.Content className="min-w-60 max-w-80">
          <Popover.Dialog>
            {isLoading ? (
              <div className="flex items-center justify-center p-4">
                <Spinner size="sm" />
              </div>
            ) : (
              <>
                {/* All joined seasons */}
                {allBalances.length > 0 && (
                  <div className="flex flex-col gap-1 max-h-[400px] overflow-y-auto overscroll-contain">
                    {allBalances.map((b) => {
                      const isCurrent = b.season_id === currentSeasonId
                      return (
                        <Button
                          key={b.id}
                          className={`flex items-center justify-between gap-4 px-3 py-2 rounded-lg w-full text-left transition-colors duration-100 hover:bg-(--color-border) ${
                            isCurrent ? '!bg-[color-mix(in_srgb,var(--color-primary)_8%,transparent)]' : 'bg-transparent'
                          }`}
                          onClick={() => navigate(`/d/${b.season_id}`)}
                        >
                          <span className="text-sm font-medium text-(--color-text) min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">{b.season.title}</span>
                          <span className="flex items-center gap-1.5 shrink-0">
                            <span className="text-[0.8125rem] font-semibold text-(--color-coin) whitespace-nowrap shrink-0">{b.balance.toLocaleString('ru-RU')} баллов</span>
                            <Check size={14} className={`text-(--color-primary) shrink-0 ${isCurrent ? 'visible' : 'invisible'}`} />
                          </span>
                        </Button>
                      )
                    })}
                  </div>
                )}

                {/* Unjoined seasons as chips */}
                {unjoinedSeasons.length > 0 && (
                  <>
                    {allBalances.length > 0 && <div className="h-px bg-(--color-border) my-1 mx-2" />}
                    <p className="text-[0.6875rem] font-semibold uppercase tracking-wider text-(--color-text-subtle) px-4 py-1">Доступные сезоны</p>
                    <div className="flex flex-col gap-1">
                      {unjoinedSeasons.map((c) => (
                        <div
                          key={c.id}
                          className="flex items-center justify-between gap-4 px-3 py-2 rounded-lg w-full text-left transition-colors duration-100 hover:bg-(--color-border) cursor-pointer"
                          onClick={isJoining ? undefined : () => handleJoin(c.id)}
                        >
                          <span className="text-sm font-medium text-(--color-text) min-w-0 overflow-hidden text-ellipsis whitespace-nowrap">{c.title}</span>
                          <Button size="sm" variant='primary' isDisabled={isJoining} onClick={isJoining ? undefined : () => handleJoin(c.id)}>
                            Начать
                          </Button>
                        </div>
                      ))}
                    </div>
                  </>
                )}

                {allBalances.length === 0 && unjoinedSeasons.length === 0 && (
                  <p className="text-sm text-(--color-text-muted) text-center py-4 px-3">Нет других сезонов</p>
                )}
              </>
            )}
          </Popover.Dialog>
        </Popover.Content>
      </Popover>
    </div>
  )
}
