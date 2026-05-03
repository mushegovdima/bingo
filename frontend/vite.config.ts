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
    tailwindcss({
      optimize: true,
    }),
    react(),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/@heroui')) return 'vendor-heroui'
          if (id.includes('node_modules/@tanstack')) return 'vendor-query'
          if (id.includes('node_modules/react-router')) return 'vendor-router'
          if (id.includes('node_modules/react') || id.includes('node_modules/react-dom')) return 'vendor-react'
        },
      },
      
    },
  },

  optimizeDeps: {
    include: ['react', 'react-dom', '@heroui/react'],
  },
})
