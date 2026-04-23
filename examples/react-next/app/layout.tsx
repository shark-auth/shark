import Providers from './providers'
export const dynamic = 'force-dynamic'
export const metadata = { title: 'Shark Auth Next example' }
export default function Root({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en"><body><Providers>{children}</Providers></body></html>
  )
}
