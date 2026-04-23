import { defineConfig } from 'tsup'

export default defineConfig({
  entry: ['src/index.ts', 'src/core/index.ts', 'src/hooks/index.ts'],
  format: ['cjs', 'esm'],
  dts: true,
  clean: true,
  splitting: false,
  external: ['react', 'react-dom'],
})
