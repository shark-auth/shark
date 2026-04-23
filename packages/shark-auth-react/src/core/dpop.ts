/**
 * RFC 9449 DPoP proof generation using browser WebCrypto API.
 */

function base64url(bytes: Uint8Array): string {
  let binary = ''
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i])
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}

async function getJkt(publicJwk: JsonWebKey): Promise<string> {
  // Canonical form for P-256 thumbprint (RFC 7638)
  const canonical = JSON.stringify({
    crv: publicJwk.crv,
    kty: publicJwk.kty,
    x: publicJwk.x,
    y: publicJwk.y,
  })
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(canonical))
  return base64url(new Uint8Array(digest))
}

async function getAth(accessToken: string): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(accessToken))
  return base64url(new Uint8Array(digest))
}

export interface DPoPProver {
  jkt: string
  createProof(method: string, url: string, accessToken?: string): Promise<string>
}

export async function generateDPoPProver(): Promise<DPoPProver> {
  const keyPair = await crypto.subtle.generateKey(
    { name: 'ECDSA', namedCurve: 'P-256' },
    true,
    ['sign'],
  )

  const publicJwk = await crypto.subtle.exportKey('jwk', keyPair.publicKey)
  const thumbprint = await getJkt(publicJwk)

  return {
    jkt: thumbprint,
    async createProof(method: string, url: string, accessToken?: string): Promise<string> {
      // RFC 9449 §4.2: Strip query and fragment
      const htu = url.split(/[?#]/)[0]
      const htm = method.toUpperCase()

      const header = {
        typ: 'dpop+jwt',
        alg: 'ES256',
        jwk: {
          kty: publicJwk.kty,
          crv: publicJwk.crv,
          x: publicJwk.x,
          y: publicJwk.y,
        },
      }

      const payload: any = {
        jti: base64url(crypto.getRandomValues(new Uint8Array(16))),
        htm,
        htu,
        iat: Math.floor(Date.now() / 1000),
      }

      if (accessToken) {
        payload.ath = await getAth(accessToken)
      }

      const encodedHeader = base64url(new TextEncoder().encode(JSON.stringify(header)))
      const encodedPayload = base64url(new TextEncoder().encode(JSON.stringify(payload)))

      const signature = await crypto.subtle.sign(
        { name: 'ECDSA', hash: { name: 'SHA-256' } },
        keyPair.privateKey,
        new TextEncoder().encode(`${encodedHeader}.${encodedPayload}`),
      )

      return `${encodedHeader}.${encodedPayload}.${base64url(new Uint8Array(signature))}`
    },
  }
}
