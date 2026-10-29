import { getAccessToken } from '../auth/oidc'

async function request<T>(path: string, init?: RequestInit, json = true): Promise<T> {
  const token = await getAccessToken()
  const headers: Record<string, string> = {}
  if (json) headers['Content-Type'] = 'application/json'
  if (token) headers.Authorization = `Bearer ${token}`
  const res = await fetch(`/api${path}`, { headers, ...init })
  if (!res.ok) {
    const body = await res.text()
    throw new Error(`${res.status} ${res.statusText}: ${body}`)
  }
  if (res.status === 204) return undefined as T
  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  postForm: <T>(path: string, body: FormData) =>
    request<T>(path, { method: 'POST', body }, false),
  download: async (path: string, filename: string) => {
    const token = await getAccessToken()
    const res = await fetch(`/api${path}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
    const url = URL.createObjectURL(await res.blob())
    const link = document.createElement('a')
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    URL.revokeObjectURL(url)
  },
  streamDownload: async (path: string, filename: string) => {
    type SavePicker = (options: {
      suggestedName: string
    }) => Promise<{ createWritable: () => Promise<WritableStream<Uint8Array>> }>
    const picker = (window as unknown as { showSaveFilePicker?: SavePicker }).showSaveFilePicker
    const authHeaders = async (): Promise<Record<string, string>> => {
      const token = await getAccessToken()
      return token ? { Authorization: `Bearer ${token}` } : {}
    }
    if (!picker) {
      const res = await fetch(`/api${path}`, { headers: await authHeaders() })
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`)
      const url = URL.createObjectURL(await res.blob())
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      link.remove()
      URL.revokeObjectURL(url)
      return
    }
    const handle = await picker({ suggestedName: filename })
    const res = await fetch(`/api${path}`, { headers: await authHeaders() })
    if (!res.ok || !res.body) throw new Error(`${res.status} ${res.statusText}`)
    const writable = await handle.createWritable()
    try {
      await res.body.pipeTo(writable)
    } catch (error) {
      await writable.abort().catch(() => undefined)
      throw error
    }
  },
}
