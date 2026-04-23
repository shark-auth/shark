import { useContext } from 'react'
import { AuthContext } from './context'

export function useUser() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useUser must be used within SharkProvider')
  return {
    isLoaded: ctx.isLoaded,
    user: ctx.user,
  }
}
