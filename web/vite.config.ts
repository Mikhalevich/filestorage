import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// API requests are proxied to the Go server (cmd/web); override its address with API_TARGET.
const apiTarget = process.env.API_TARGET ?? 'http://localhost:8080'

const proxy = {
  '/auth/': apiTarget,
  '/files/': apiTarget,
  '/download/': apiTarget,
  '/dirs/': apiTarget,
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: { proxy },
  preview: { proxy },
})
