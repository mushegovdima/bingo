import { useMutation, useQueryClient } from '@tanstack/react-query'
import { submissionsApi } from '@/lib/api/submissions'

export function useApproveSubmission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (submissionId: number) => submissionsApi.approve(submissionId),
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: ['submissions'] })
      queryClient.refetchQueries({ queryKey: ['submissions-admin'] })
    },
  })
}

export function useRejectSubmission() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ id, comment }: { id: number; comment: string }) =>
      submissionsApi.reject(id, comment),
    onSuccess: () => {
      queryClient.refetchQueries({ queryKey: ['submissions'] })
      queryClient.refetchQueries({ queryKey: ['submissions-admin'] })
    },
  })
}
