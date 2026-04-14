import { useQuery } from '@tanstack/react-query'
import { tasksApi } from '@/lib/api/tasks'

export function useSeasonTasks(seasonId: number) {
  return useQuery({
    queryKey: ['tasks', 'season', seasonId],
    queryFn: () => tasksApi.listBySeason(seasonId),
    enabled: seasonId > 0,
  })
}
