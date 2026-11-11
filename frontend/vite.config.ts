import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// In development the browser only talks to Vite (localhost:5173). Vite
// forwards API paths to the Go API, so for the browser everything is
// same-origin: no CORS setup, and cookies (Phase 1 sessions) just work.
const API_TARGET = 'http://localhost:8081'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      '/api': { target: API_TARGET, changeOrigin: true },
      // Exact matches (regex keys start with ^), so a future page like
      // /ready-to-ship is not accidentally sent to the API.
      '^/(health|ready)$': { target: API_TARGET, changeOrigin: true },
    },
  },
})
