import { defineSetupVue3 } from '@histoire/plugin-vue'
import { Quasar } from 'quasar'
import quasarLang from 'quasar/lang/pt-BR'

import '@fontsource/inter/index.css'
import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/src/css/index.sass'
import './css/app.scss'

if (typeof window !== 'undefined' && !window.matchMedia) {
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: () => {}, // deprecated
      removeListener: () => {}, // deprecated
      addEventListener: () => {},
      removeEventListener: () => {},
      dispatchEvent: () => false,
    }),
  })
}

import { createPinia } from 'pinia'

export const setupVue3 = defineSetupVue3(({ app }) => {
  app.use(createPinia())
  app.use(Quasar, {
    config: {
      dark: false,
    },
    lang: quasarLang,
  })
})
