# index.ts

**Path:** `admin/src/design/index.ts`
**Type:** Design system exports
**LOC:** 24

## Purpose
Central export file for design system—re-exports all primitives, tokens, and types.

## Exports
- `tokens` — design system constants
- `Button`, `ButtonProps`, `ButtonVariant`, `ButtonSize`
- `Input`, `InputProps`
- `FormField`, `FormFieldProps`
- `Card`, `CardProps`
- `Modal`, `ModalProps`
- `Tabs`, `TabsProps`, `Tab`
- `ToastProvider`, `ToastContext`, `useToast`, `ToastProviderProps`, `ToastContextValue`, `ToastItem`, `ToastVariant`

## Usage
```typescript
import { Button, tokens, useToast } from '@/design'
```

## Notes
- Single import path for all design primitives
- Includes both components and types
