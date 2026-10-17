/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

// The client build of Quasar, for tests that mount real Quasar components:
// under Vitest the plain 'quasar' entry resolves to the SSR build.
declare module 'quasar/dist/quasar.client.js' {
  export * from 'quasar'
}
