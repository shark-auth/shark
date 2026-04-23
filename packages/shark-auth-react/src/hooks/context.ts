import { createContext } from 'react'
import type { User, Session, Organization } from '../core/types'
import type { createClient } from '../core/client'

export interface AuthContextValue {
  isLoaded: boolean
  isAuthenticated: boolean
  user: User | null
  session: Session | null
  organization: Organization | null
  client: ReturnType<typeof createClient>
  signOut: () => Promise<void>
  /** Available for internal use by SignIn/SignUp components */
  authUrl?: string
  publishableKey?: string
}

export const AuthContext = createContext<AuthContextValue | null>(null)
