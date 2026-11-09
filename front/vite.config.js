import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const authTarget = new URL(env.AUTH_PROXY_TARGET || 'https://toko.azlaundry.my.id')
  return {
    plugins: [vue()],
    base: env.VITE_BASE_PATH || '/',
    server: {
      port: 5173,
      proxy: {
        '/api/v1': {
          target: authTarget.origin,
          changeOrigin: true,
          ws: true,
          configure(proxy) {
            const rewriteOrigin = (outgoing, incoming) => {
              // Translate only a same-origin development request; preserve hostile origins for Go to reject.
              const origin = incoming.headers.origin
              if (
                origin === `http://${incoming.headers.host}` ||
                origin === `https://${incoming.headers.host}`
              )
                outgoing.setHeader('Origin', authTarget.origin)
            }
            proxy.on('proxyReq', rewriteOrigin)
            proxy.on('proxyReqWs', rewriteOrigin)
            proxy.on('proxyRes', (response, incoming) => {
              const hostname = new URL(`http://${incoming.headers.host}`).hostname
              if (
                ['127.0.0.1', 'localhost', '[::1]'].includes(hostname) &&
                response.headers['set-cookie']
              )
                // Live cookies stay Secure on HTTPS. Loopback Vite uses HTTP during local development.
                response.headers['set-cookie'] = response.headers['set-cookie'].map((cookie) =>
                  cookie.replace(/;\s*Secure\b/gi, ''),
                )
            })
          },
        },
      },
    },
    build: { outDir: 'dist' },
  }
})
