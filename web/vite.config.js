import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue2'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 8088,
    proxy: {
      '/api': {
        target: 'http://localhost:8087',
        changeOrigin: true
      }
    }
  }
})
