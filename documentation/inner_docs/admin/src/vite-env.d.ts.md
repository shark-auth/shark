# vite-env.d.ts

**Path:** `admin/src/vite-env.d.ts`
**Type:** Type declarations
**LOC:** 22

## Purpose
Vite client type definitions + image/asset module declarations for TypeScript.

## Exports
Module declarations for:
- `*.png` → string (data URL)
- `*.jpg` → string
- `*.jpeg` → string
- `*.svg` → string

## Notes
- Extends Vite client types for build asset importing
- Allows `import logo from './img.png'` syntax with proper typing
