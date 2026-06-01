import { defineConfig } from 'vitest/config'
import { fileURLToPath, URL } from 'node:url'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
    plugins: [vue()],
    resolve: {
        alias: {
            '@': fileURLToPath(new URL('./src', import.meta.url)),
        },
    },
    test: {
        environment: 'happy-dom',
        setupFiles: ['./src/test/setup.js'],
        include: ['src/**/*.{test,spec}.{js,ts}'],
        globals: true,
        coverage: {
            provider: 'v8',
            reporter: ['text', 'lcov'],
            include: ['src/**/*.{js,vue}'],
            exclude: ['src/test/**', 'src/**/*.test.js'],
        },
    },
})
