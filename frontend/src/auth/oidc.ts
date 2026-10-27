import { UserManager, WebStorageStateStore } from 'oidc-client-ts'

const authority = import.meta.env.VITE_OIDC_AUTHORITY ?? ''
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID ?? 'fai'

// Covers both local dev (VITE_DISABLE_AUTH=true) and the docker-compose deployment:
// there, oauth2-proxy + Caddy's forward_auth gate every request — including the
// initial page load — before the browser ever reaches this SPA, so no authority is
// baked in at build time and the frontend correctly does no client-side auth of its
// own (a second login flow here would conflict with the one oauth2-proxy already
// completed). The kubernetes deployment's build bakes in a real authority instead.
export const authDisabled = import.meta.env.VITE_DISABLE_AUTH === 'true' || !authority

export const userManager = authDisabled
  ? null
  : new UserManager({
      authority,
      client_id: clientId,
      redirect_uri: `${window.location.origin}/auth/callback`,
      post_logout_redirect_uri: window.location.origin,
      response_type: 'code',
      scope: 'openid profile email',
      userStore: new WebStorageStateStore({ store: window.localStorage }),
      automaticSilentRenew: true,
    })

export async function getAccessToken(): Promise<string | null> {
  if (!userManager) return null
  const user = await userManager.getUser()
  return user?.access_token ?? null
}

export async function login() {
  if (!userManager) return
  await userManager.signinRedirect()
}

export async function logout() {
  if (!userManager) return
  await userManager.signoutRedirect()
}
