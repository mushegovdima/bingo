// Minimal Telegram Web App SDK types
interface TelegramWebAppUser {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  language_code?: string
}

interface TelegramWebApp {
  initData: string
  initDataUnsafe: {
    user?: TelegramWebAppUser
    auth_date: number
    hash: string
    query_id?: string
  }
  ready(): void
  expand(): void
  close(): void
  isExpanded: boolean
  platform: string
}

interface Window {
  Telegram?: {
    WebApp?: TelegramWebApp
  }
}
