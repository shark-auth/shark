import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { resolve } from 'path';
export default defineConfig({
    plugins: [react()],
    base: '/admin/',
    build: {
        outDir: '../internal/admin/dist',
        emptyOutDir: true,
        rollupOptions: {
            input: {
                main: resolve(__dirname, 'index.html'),
                hosted: resolve(__dirname, 'hosted.html'),
            },
            output: {
                entryFileNames: (c) => c.name === 'hosted'
                    ? 'hosted/assets/[name]-[hash].js'
                    : 'assets/[name]-[hash].js',
                assetFileNames: (i) => i.name?.startsWith('hosted')
                    ? 'hosted/assets/[name]-[hash][extname]'
                    : 'assets/[name]-[hash][extname]',
                chunkFileNames: (c) => c.name?.startsWith('hosted')
                    ? 'hosted/assets/[name]-[hash].js'
                    : 'assets/[name]-[hash].js',
            },
        },
    },
    server: {
        proxy: {
            '/api': 'http://localhost:8080',
            '/.well-known': 'http://localhost:8080',
        }
    }
});
