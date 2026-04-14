

import { AppNavbar } from './AppNavbar'

interface Props {
  children: React.ReactNode
}

/**
 * Wraps every authenticated page with the top navbar and a centered content container.
 */
export function AppLayout({ children }: Props) {
  return (
    <div className="app-layout">
      <AppNavbar />
      <main className="flex-1 w-full max-w-6xl mx-auto px-3 py-3">
        {children}
      </main>
    </div>
  )
}
