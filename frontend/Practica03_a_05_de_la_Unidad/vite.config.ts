import tailwindcss from '@tailwindcss/vite'
import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'
import { VitePWA } from 'vite-plugin-pwa'

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    VitePWA({
      registerType: 'prompt',
      injectRegister: false,
      includeAssets: ['favicon.svg', 'apple-touch-icon.png'],
      manifest: {
        name: 'Dentalia - Gestión Odontológica',
        short_name: 'Dentalia',
        description: 'Sistema de demostración para la gestión de una clínica odontológica',
        lang: 'es-EC',
        theme_color: '#064e3b',
        background_color: '#f8fafc',
        display: 'standalone',
        scope: '/',
        start_url: '/',
        icons: [
          {
            src: '/pwa-192x192.png',
            sizes: '192x192',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: '/pwa-512x512.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'any',
          },
          {
            src: '/pwa-512x512-maskable.png',
            sizes: '512x512',
            type: 'image/png',
            purpose: 'maskable',
          },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,png,svg,webmanifest,woff,woff2,ttf,eot}'],
        navigateFallback: '/index.html',
        navigateFallbackDenylist: [/^\/api\//],
        // No se cachean APIs: el expediente y la información clínica son datos sensibles.
        runtimeCaching: [],
      },
    }),
  ],
  server: {
    host: true,
  },
})
