import { useNavigate } from 'react-router-dom'
import { SeasonList } from './tabs/season/SeasonList'

export default function AdminSeasonsPage() {
  const navigate = useNavigate()
  return (
    <SeasonList
      selectedId={null}
      onSelect={(id) => navigate(`/admin/seasons/${id}`)}
    />
  )
}
