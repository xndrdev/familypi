// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  // Reine SPA: das Go-Backend liefert die generierten Dateien aus.
  ssr: false,
  devtools: { enabled: false },
  css: ['~/assets/css/base.css'],
  app: {
    head: {
      title: 'Familie',
      htmlAttrs: { lang: 'de' },
      meta: [{ name: 'viewport', content: 'width=device-width, initial-scale=1' }],
    },
  },
  nitro: {
    // Im Dev-Modus laeuft das Go-Backend separat auf :8080.
    devProxy: {
      '/api': { target: 'http://localhost:8080/api', changeOrigin: true },
      '/calendar.ics': { target: 'http://localhost:8080/calendar.ics', changeOrigin: true },
    },
  },
})
