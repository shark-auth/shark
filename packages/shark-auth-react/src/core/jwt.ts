import { jwtVerify, createRemoteJWKSet } from 'jose'
import type { JWTPayload } from 'jose'

function base64urlDecode(str: string): string {
  // Pad to multiple of 4
  const padded = str + '='.repeat((4 - (str.length % 4)) % 4)
  const standard = padded.replace(/-/g, '+').replace(/_/g, '/')
  return atob(standard)
}

export interface SharkClaims extends JWTPayload {
  sub?: string
  email?: string
  firstName?: string
  lastName?: string
  orgId?: string
  [key: string]: unknown
}

export function decodeClaims(token: string): SharkClaims {
  const parts = token.split('.')
  if (parts.length !== 3) throw new Error('Invalid JWT format')
  try {
    const payload = base64urlDecode(parts[1])
    return JSON.parse(payload) as SharkClaims
  } catch (err) {
    throw new Error(`Failed to decode JWT payload: ${String(err)}`)
  }
}

export async function verifyToken(token: string, jwksUrl: string): Promise<JWTPayload> {
  const JWKS = createRemoteJWKSet(new URL(jwksUrl))
  const { payload } = await jwtVerify(token, JWKS)
  return payload
}
