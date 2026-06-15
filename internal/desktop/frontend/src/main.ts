import { createApp } from 'vue'
import { Quasar, Notify, Dialog } from 'quasar'
import quasarLang from 'quasar/lang/pt-BR'
import quasarIconSet from 'quasar/icon-set/material-icons'

import '@fontsource/inter/index.css'
import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/src/css/index.sass'

import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import { useAppStore } from './stores/app'

const myApp = createApp(App)
const pinia = createPinia()

myApp.use(pinia)
myApp.use(router)

myApp.use(Quasar, {
  plugins: {
    Notify,
    Dialog,
  },
  config: {
    dark: 'auto',
  },
  lang: quasarLang,
  iconSet: quasarIconSet,
})

useAppStore(pinia).initLogListeners()

myApp.mount('#app')
