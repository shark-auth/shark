const isBrowser = typeof window !== 'undefined'

const KEYS = {
  ACCESS_TOKEN: 'shark_access_token',
  ACCESS_TOKEN_EXPIRES_AT: 'shark_access_token_expires_at',
  REFRESH_TOKEN: 'shark_refresh_token',
  CODE_VERIFIER: 'shark_code_verifier',
  REDIRECT_AFTER: 'shark_redirect_after',
} as const

function get(key: string): string | null {
  if (!isBrowser) return null
  try {
    return sessionStorage.getItem(key)
  } catch {
    return null
  }
}

function set(key: string, value: string): void {
  if (!isBrowser) return
  try {
    sessionStorage.setItem(key, value)
  } catch {
    // storage may be unavailable (private mode quota, etc.)
  }
}

function remove(key: string): void {
  if (!isBrowser) return
  try {
    sessionStorage.removeItem(key)
  } catch {
    // ignore
  }
}

export function getAccessToken(): string | null {
  const token = get(KEYS.ACCESS_TOKEN)
  const expiresAt = get(KEYS.ACCESS_TOKEN_EXPIRES_AT)
  if (!token || !expiresAt) return null
  // Return null if expired (allow caller to handle refresh)
  if (Date.now() > Number(expiresAt)) return null
  return token
}

export function getRefreshToken(): string | null {
  return get(KEYS.REFRESH_TOKEN)
}

export function setAccessToken(token: string, expiresAt: number, refreshToken?: string): void {
  set(KEYS.ACCESS_TOKEN, token)
  set(KEYS.ACCESS_TOKEN_EXPIRES_AT, String(expiresAt))
  if (refreshToken) set(KEYS.REFRESH_TOKEN, refreshToken)
}

export function clearAll(): void {
  if (!isBrowser) return
  Object.values(KEYS).forEach(remove)
}

export function getCodeVerifier(): string | null {
  return get(KEYS.CODE_VERIFIER)
}

export function setCodeVerifier(verifier: string): void {
  set(KEYS.CODE_VERIFIER, verifier)
}

export function getRedirectAfter(): string | null {
  return get(KEYS.REDIRECT_AFTER)
}

export function setRedirectAfter(url: string): void {
  set(KEYS.REDIRECT_AFTER, url)
}
