import { useContext } from 'react'
import { AuthContext } from './context'

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within SharkProvider')
  return {
    isLoaded: ctx.isLoaded,
    isAuthenticated: ctx.isAuthenticated,
    signOut: ctx.signOut,
  }
}
