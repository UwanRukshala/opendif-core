import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// Port 3002 matches the eSignet redirect URI (http://localhost:3002).
// Vite default 5173 is often blocked on Windows (reserved range 5141–5240).
const devPort = Number(process.env.PORT) || 3002

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    host: 'localhost',
    port: devPort,
    strictPort: true,
  },
  preview: {
    host: 'localhost',
    port: devPort,
    strictPort: true,
  },
})
