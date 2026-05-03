import { useQuery } from '@tanstack/react-query'
import { templatesApi } from '@/lib/api/templates'

export function useTemplates() {
  return useQuery({
    queryKey: ['templates'],
    queryFn: () => templatesApi.list(),
  })
}

export function useTemplateHistory(codename: string | null) {
  return useQuery({
    queryKey: ['templates', codename, 'history'],
    queryFn: () => templatesApi.listHistory(codename!),
    enabled: !!codename,
  })
}
