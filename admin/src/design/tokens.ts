export const tokens = {
  color: {
    primary: 'var(--shark-primary, #7c3aed)',
    primaryFg: '#ffffff',
    surface0: 'oklch(10% 0.005 256)',
    surface1: 'oklch(14% 0.005 256)',
    surface2: 'oklch(18% 0.005 256)',
    surface3: 'oklch(22% 0.005 256)',
    hairline: 'oklch(28% 0.005 256)',
    fg: 'oklch(98% 0.005 256)',
    fgMuted: 'oklch(70% 0.005 256)',
    fgDim: 'oklch(50% 0.005 256)',
    danger: 'oklch(62% 0.2 25)',
    dangerFg: '#ffffff',
    warn: 'oklch(74% 0.16 85)',
    success: 'oklch(68% 0.15 160)',
    focusRing: 'var(--shark-primary, #7c3aed)',
  },
  space: { 1: 4, 2: 8, 3: 12, 4: 16, 6: 24, 8: 32, 12: 48 },
  radius: { sm: 2, md: 2, lg: 2, xl: 4 },
  type: {
    display: { family: 'var(--font-display, Manrope), -apple-system, sans-serif' },
    body: { family: 'var(--font-body, Manrope), sans-serif' },
    mono: { family: 'var(--font-mono, Azeret Mono), monospace' },
    size: { xs: 11, sm: 12, base: 14, md: 16, lg: 18, xl: 22, '2xl': 28 },
    weight: { regular: 400, medium: 500, semibold: 600, bold: 700 },
  },
  motion: {
    fast: '60ms cubic-bezier(0.2, 0, 0, 1)',
    med: '120ms cubic-bezier(0.2, 0, 0, 1)',
  },
  shadow: {
    sm: '0 1px 2px oklch(0% 0 0 / 20%)',
    md: '0 4px 12px oklch(0% 0 0 / 28%)',
  },
  zIndex: { dropdown: 100, modal: 1000, toast: 2000 },
} as const

export type Tokens = typeof tokens
