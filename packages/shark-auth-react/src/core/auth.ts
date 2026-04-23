import { setCodeVerifier, getCodeVerifier, setRedirectAfter, setAccessToken } from './storage'

function base64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let str = ''
  for (const b of bytes) str += String.fromCharCode(b)
  return btoa(str).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

export function generateCodeVerifier(): string {
  const bytes = new Uint8Array(48) // 48 bytes → 64 base64url chars
  crypto.getRandomValues(bytes)
  return base64url(bytes.buffer)
}

export async function generateCodeChallenge(verifier: string): Promise<string> {
  const encoded = new TextEncoder().encode(verifier)
  const digest = await crypto.subtle.digest('SHA-256', encoded)
  return base64url(digest)
}

function randomState(): string {
  const bytes = new Uint8Array(16)
  crypto.getRandomValues(bytes)
  return base64url(bytes.buffer)
}

export interface AuthFlowResult {
  url: string
  state: string
}

export async function startAuthFlow(
  redirectUrl: string,
  authUrl: string,
  publishableKey: string,
): Promise<AuthFlowResult> {
  const verifier = generateCodeVerifier()
  const challenge = await generateCodeChallenge(verifier)
  const state = randomState()

  setCodeVerifier(verifier)
  setRedirectAfter(redirectUrl)

  const base = authUrl.replace(/\/$/, '')
  const callbackUri = `${typeof window !== 'undefined' ? window.location.origin : ''}/shark/callback`

  const params = new URLSearchParams({
    response_type: 'code',
    client_id: publishableKey,
    redirect_uri: callbackUri,
    code_challenge: challenge,
    code_challenge_method: 'S256',
    state,
  })

  return { url: `${base}/oauth/authorize?${params.toString()}`, state }
}

export interface TokenResponse {
  access_token: string
  refresh_token?: string
  expires_in?: number
  expires_at?: number
}

export interface ExchangeOptions {
  code?: string
  refreshToken?: string
  dpopProof?: string
}

export async function exchangeToken(
  authUrl: string,
  publishableKey: string,
  opts: ExchangeOptions,
): Promise<TokenResponse> {
  const base = authUrl.replace(/\/$/, '')
  const callbackUri = `${typeof window !== 'undefined' ? window.location.origin : ''}/shark/callback`

  const body = new URLSearchParams({
    client_id: publishableKey,
  })

  if (opts.code) {
    const verifier = getCodeVerifier()
    if (!verifier) throw new Error('No code verifier found in storage')
    body.set('grant_type', 'authorization_code')
    body.set('code', opts.code)
    body.set('code_verifier', verifier)
    body.set('redirect_uri', callbackUri)
  } else if (opts.refreshToken) {
    body.set('grant_type', 'refresh_token')
    body.set('refresh_token', opts.refreshToken)
  } else {
    throw new Error('Either code or refreshToken is required')
  }

  const headers = new Headers({
    'Content-Type': 'application/x-www-form-urlencoded',
  })
  if (opts.dpopProof) {
    headers.set('DPoP', opts.dpopProof)
  }

  const resp = await fetch(`${base}/oauth/token`, {
    method: 'POST',
    headers,
    body: body.toString(),
  })

  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(`Token exchange failed ${resp.status}: ${text}`)
  }

  const data: TokenResponse = await resp.json()
  const expiresAt = data.expires_at
    ? data.expires_at * 1000
    : Date.now() + (data.expires_in ?? 3600) * 1000

  setAccessToken(data.access_token, expiresAt, data.refresh_token)
  return data
}

/** @deprecated Use exchangeToken */
export async function exchangeCodeForToken(
  code: string,
  authUrl: string,
  publishableKey: string,
): Promise<TokenResponse> {
  return exchangeToken(authUrl, publishableKey, { code })
}

