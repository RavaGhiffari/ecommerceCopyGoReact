import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// In development the browser only talks to Vite. Vite forwards API paths to
// the Go API, so for the browser everything is same-origin: the session cookie
// is a normal first-party cookie and no CORS setup is needed.
const API_TARGET = 'http://127.0.0.1:8081'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    // "@/components/ui/button" → src/components/ui/button (same alias as tsconfig).
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      // changeOrigin: false keeps the browser's Host header (e.g. localhost:5174).
      // The API's CSRF check compares the Origin header with Host; rewriting
      // Host to localhost:8081 would make every same-site POST look cross-site.
      '/api': { target: API_TARGET, changeOrigin: false },
      // Exact matches (regex keys start with ^), so a future page like
      // /ready-to-ship is not accidentally sent to the API.
      '^/(health|ready)$': { target: API_TARGET, changeOrigin: true },
    },
  },
})
