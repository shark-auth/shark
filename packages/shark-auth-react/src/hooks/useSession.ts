import { useContext } from 'react'
import { AuthContext } from './context'

export function useSession() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useSession must be used within SharkProvider')
  return {
    isLoaded: ctx.isLoaded,
    session: ctx.session,
  }
}
