import { SeasonList } from './season/SeasonList'
import { SeasonDetail } from './season/SeasonDetail'

interface Props {
  seasonId: number | null
  onSelect: (id: number) => void
}

export function SeasonTab({ seasonId, onSelect }: Props) {
  if (seasonId !== null) {
    return <SeasonDetail seasonId={seasonId} />
  }

  return <SeasonList selectedId={seasonId} onSelect={onSelect} />
}
