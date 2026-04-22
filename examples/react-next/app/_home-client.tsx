'use client'
import { SignIn, UserButton, SignedIn, SignedOut } from '@shark-auth/react'
export default function HomeClient() {
  return (
    <main style={{ padding: 48 }}>
      <h1>Shark Auth — Next.js example</h1>
      <SignedOut><SignIn redirectUrl="/dashboard"/></SignedOut>
      <SignedIn><UserButton/></SignedIn>
    </main>
  )
}
