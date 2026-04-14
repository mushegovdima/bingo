import { useParams } from 'react-router-dom'
import { SeasonDetail } from './tabs/season/SeasonDetail'

export default function AdminSeasonDetailPage() {
  const { seasonId } = useParams<{ seasonId: string }>()

  return (
    <SeasonDetail seasonId={Number(seasonId)} />
  )
}
