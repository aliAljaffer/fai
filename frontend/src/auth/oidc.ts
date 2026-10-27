import { UserManager, WebStorageStateStore } from 'oidc-client-ts'

export const authDisabled = import.meta.env.VITE_DISABLE_AUTH === 'true'

const authority = import.meta.env.VITE_OIDC_AUTHORITY ?? ''
const clientId = import.meta.env.VITE_OIDC_CLIENT_ID ?? 'fai'

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
