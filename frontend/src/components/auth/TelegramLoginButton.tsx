

import { useEffect, useRef } from 'react'
import { TelegramAuthData } from '@/types'

interface Props {
  onAuth: (data: TelegramAuthData) => void
}

export function TelegramLoginButton({ onAuth }: Props) {
  const containerRef = useRef<HTMLDivElement>(null)
  const botName = import.meta.env.VITE_TG_BOT_NAME ?? ''

  useEffect(() => {
    if (!botName || !containerRef.current) return

    const callbackName = '__tgAuthCallback'
    ;(window as unknown as Record<string, unknown>)[callbackName] = onAuth

    const script = document.createElement('script')
    script.src = 'https://telegram.org/js/telegram-widget.js?22'
    script.setAttribute('data-telegram-login', botName)
    script.setAttribute('data-size', 'large')
    script.setAttribute('data-radius', '8')
    script.setAttribute('data-onauth', `${callbackName}(user)`)
    script.setAttribute('data-request-access', 'write')
    script.async = true

    containerRef.current.innerHTML = ''
    containerRef.current.appendChild(script)

    return () => {
      delete (window as unknown as Record<string, unknown>)[callbackName]
    }
  }, [botName, onAuth])

  if (!botName) {
    return (
      <p className="text-red-500 text-sm">
        VITE_TG_BOT_NAME не задан в .env.local
      </p>
    )
  }

  return <div ref={containerRef} />
}
