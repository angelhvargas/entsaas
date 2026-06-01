import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
    plugins: [
        tailwindcss(),
        vue(),
    ],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url))
        }
    },
    server: {
        host: '0.0.0.0',
        port: 5173,
        strictPort: true,
        proxy: {
            '/v1': {
                target: process.env.API_TARGET || 'http://localhost:8080',
                changeOrigin: true,
                configure: (proxy) => {
                    proxy.on('proxyReq', (proxyReq, req) => {
                        if (req.headers.authorization) {
                            proxyReq.setHeader('Authorization', req.headers.authorization)
                        }
                    })
                }
            }
        }
    },
    build: {
        outDir: 'dist',
        emptyOutDir: true,
        chunkSizeWarningLimit: 700,
        rolldownOptions: {
            output: {
                entryFileNames: 'assets/[name]-[hash].js',
                chunkFileNames: 'assets/[name]-[hash].js',
                assetFileNames: 'assets/[name]-[hash].[ext]',
                manualChunks(id) {
                    if (id.includes('node_modules/vue') || id.includes('node_modules/@vue')) {
                        return 'vendor-vue'
                    }
                    if (id.includes('node_modules/lucide')) {
                        return 'vendor-icons'
                    }
                }
            }
        }
    }
})
