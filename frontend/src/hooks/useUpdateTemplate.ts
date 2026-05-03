import { useMutation, useQueryClient } from '@tanstack/react-query'
import { templatesApi } from '@/lib/api/templates'

export function useUpdateTemplate() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ codename, body }: { codename: string; body: string }) =>
      templatesApi.updateBody(codename, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
    },
  })
}
