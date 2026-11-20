import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { quasar, transformAssetUrls } from '@quasar/vite-plugin'

// https://vitejs.dev/config/
export default defineConfig({
  // The Wails v3 dev proxy dials localhost over IPv4 only, while Vite's
  // default "localhost" may bind to ::1 alone on Windows.
  server: {
    host: '127.0.0.1',
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  // No @wailsio/runtime/plugins/vite: it injects the generated eventcreate.ts,
  // which builds event payloads only for class bindings, and ours are
  // interfaces (wails3 generate bindings -i).
  plugins: [
    vue({
      template: { transformAssetUrls },
    }),

    // @quasar/plugin-vite options list:
    // https://github.com/quasarframework/quasar/blob/dev/vite-plugin/index.d.ts
    quasar({
      sassVariables: fileURLToPath(new URL('./src/quasar-variables.sass', import.meta.url)),
    }),
  ],
})
