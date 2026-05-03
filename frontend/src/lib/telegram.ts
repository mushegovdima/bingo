
// Returns the Telegram WebApp instance if the app is running inside Telegram Mini App context.
export function getTelegramWebApp(): TelegramWebApp | null {
  return window.Telegram?.WebApp?.initData ? (window.Telegram.WebApp ?? null) : null
}

export function isTelegramMiniApp(): boolean {
  return !!getTelegramWebApp()
}
