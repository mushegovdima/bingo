import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'path'

const PORT = Number(process.env.PORT) || 3000

// https://vitejs.dev/config/
export default defineConfig({
  server: {
    port: PORT,
    allowedHosts: true,
  },
  preview: { port: PORT },
  plugins: [
    tailwindcss(),
    react(),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
})
