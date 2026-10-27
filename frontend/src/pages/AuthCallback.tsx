import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { userManager } from '../auth/oidc'

export function AuthCallback() {
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!userManager) {
      navigate('/')
      return
    }
    userManager
      .signinRedirectCallback()
      .then(() => navigate('/'))
      .catch((err) => setError(err.message))
  }, [navigate])

  if (error) return <div className="p-6 text-destructive">Login failed: {error}</div>
  return <div className="p-6 text-muted-foreground">Signing in…</div>
}
