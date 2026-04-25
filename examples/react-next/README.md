# react-next-example

Next.js 14 App Router example consuming `@sharkauth/react`.

## Setup

1. Copy `.env.example` to `.env.local` and fill in your values:
   ```
   cp .env.example .env.local
   ```

2. In the Shark admin, register your app in **Components mode** with callback URL:
   ```
   http://localhost:3000/shark/callback
   ```

3. Run the dev server from the monorepo root:
   ```
   pnpm --filter react-next-example dev
   ```

   Or from this directory:
   ```
   pnpm dev
   ```

The app will be available at http://localhost:3000.
