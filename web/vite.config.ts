import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  // Base path must match the server's /ui/ prefix so all asset paths
  // (JS, CSS, fonts) are resolved correctly when served from /ui/.
  base: '/ui/',
})
