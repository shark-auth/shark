import { getAccessToken } from './storage'

export interface SharkClient {
  fetch(path: string, init?: RequestInit): Promise<Response>
}

export function createClient(authUrl: string, publishableKey: string): SharkClient {
  const base = authUrl.replace(/\/$/, '')

  return {
    async fetch(path: string, init: RequestInit = {}): Promise<Response> {
      const url = `${base}${path.startsWith('/') ? path : '/' + path}`

      const headers = new Headers(init.headers)
      headers.set('X-Shark-Publishable-Key', publishableKey)

      const token = getAccessToken()
      if (token) {
        headers.set('Authorization', `Bearer ${token}`)
      }

      return fetch(url, { ...init, headers })
    },
  }
}
