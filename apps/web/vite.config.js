import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    port: 3000,
    proxy: {
      '/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/bank-accounts': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/categories': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/transactions': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
