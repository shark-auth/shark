import { describe, it, expect } from 'vitest'
import { decodeClaims } from '../jwt'

// hand-crafted JWT: header.payload.sig
// header: {"alg":"HS256","typ":"JWT"}
// payload: {"sub":"usr_123","email":"t@example.com"}
const HEADER = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
const PAYLOAD = btoa(JSON.stringify({ sub: 'usr_123', email: 't@example.com' }))
  .replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
const FAKE_JWT = `${HEADER}.${PAYLOAD}.fakesig`

describe('decodeClaims', () => {
  it('decodes a valid JWT payload without verification', () => {
    const claims = decodeClaims(FAKE_JWT)
    expect(claims.sub).toBe('usr_123')
    expect(claims.email).toBe('t@example.com')
  })

  it('throws on malformed token (wrong number of parts)', () => {
    expect(() => decodeClaims('not.a.valid.jwt.token')).toThrow()
  })

  it('throws on token with invalid base64url payload', () => {
    expect(() => decodeClaims('header.!!!.sig')).toThrow()
  })
})
