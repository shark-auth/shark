'use client'
import { SignedIn, SignedOut, useUser } from '@sharkauth/react'
export default function DashboardClient() {
  const { user } = useUser()
  return (
    <main style={{ padding: 48 }}>
      <SignedIn><h1>Hello {user?.email}</h1></SignedIn>
      <SignedOut><p>Please <a href="/">sign in</a></p></SignedOut>
    </main>
  )
}
