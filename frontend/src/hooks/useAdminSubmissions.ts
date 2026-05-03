import { useQuery } from '@tanstack/react-query'
import { submissionsApi } from '@/lib/api/submissions'
import { useSeasonTasks } from './useSeasonTasks'

export function useAdminSubmissions(seasonId: number | undefined) {
  const { data: tasks = [] } = useSeasonTasks(seasonId ?? 0)

  return useQuery({
    queryKey: ['submissions-admin', seasonId],
    queryFn: () => submissionsApi.getAllAdmin(),
    enabled: !!seasonId,
    select: (submissions) => {
      const taskIds = new Set(tasks.map((t) => t.id))
      return submissions.filter((s) => taskIds.has(s.task_id))
    },
  })
}
