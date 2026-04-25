'use client'
import { SharkProvider } from '@sharkauth/react'
export default function Providers({ children }: { children: React.ReactNode }) {
  return (
    <SharkProvider
      publishableKey={process.env.NEXT_PUBLIC_SHARK_KEY!}
      authUrl={process.env.NEXT_PUBLIC_SHARK_URL!}
    >
      {children}
    </SharkProvider>
  )
}
