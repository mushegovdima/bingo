

import { useQuery } from '@tanstack/react-query'
import { submissionsApi } from '@/lib/api/submissions'
import { useSeasonTasks } from './useSeasonTasks'

export function useSubmissions(seasonId: number | undefined) {
  const { data: tasks = [] } = useSeasonTasks(seasonId ?? 0)

  return useQuery({
    queryKey: ['submissions', seasonId],
    queryFn: () => submissionsApi.getAll(),
    enabled: !!seasonId,
    select: (submissions) => {
      const taskIds = new Set(tasks.map((t) => t.id))
      const forSeason = submissions.filter((s) => taskIds.has(s.task_id))
      return {
        all: forSeason,
        approvedCount: forSeason.filter((s) => s.status === 'approved').length,
        totalCount: forSeason.length,
      }
    },
  })
}
