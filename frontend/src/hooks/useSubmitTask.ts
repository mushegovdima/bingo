
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { submissionsApi, SubmitTaskRequest } from '@/lib/api/submissions'

export function useSubmitTask() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (req: SubmitTaskRequest) => submissionsApi.submit(req),
    onSuccess: () => {
      // Use refetchQueries instead of invalidateQueries so the cache is
      // updated immediately — even before any component subscribes to the
      // query on the next screen.
      queryClient.refetchQueries({ queryKey: ['submissions'] })
    },
  })
}
