import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { App } from './App'
import { TelegramWebAppProvider } from '@/components/auth/TelegramWebAppProvider'
import './index.css'
import './styles/app.scss'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      gcTime: 5 * 60_000,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <QueryClientProvider client={queryClient}>
        <TelegramWebAppProvider>
          <App />
        </TelegramWebAppProvider>
      </QueryClientProvider>
    </BrowserRouter>
  </StrictMode>,
)
