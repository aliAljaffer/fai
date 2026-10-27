import { useEffect, useState } from 'react'
import { authDisabled, login, userManager } from './oidc'

// Blocks rendering of children until a session exists, redirecting to Keycloak
// otherwise. No-ops entirely when VITE_DISABLE_AUTH=true (local dev).
export function AuthGate({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(authDisabled)

  useEffect(() => {
    if (authDisabled || !userManager) return
    if (window.location.pathname === '/auth/callback') {
      setReady(true)
      return
    }
    userManager.getUser().then((user) => {
      if (user && !user.expired) {
        setReady(true)
      } else {
        login()
      }
    })
  }, [])

  if (!ready) return <div className="p-6 text-muted-foreground">Redirecting to login…</div>
  return <>{children}</>
}
