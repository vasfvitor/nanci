import { createApp } from 'vue'
import { Quasar, Notify, Dialog } from 'quasar'
import quasarLang from 'quasar/lang/pt-BR'
import quasarIconSet from 'quasar/icon-set/material-icons-round'

import '@quasar/extras/roboto-font/roboto-font.css'
import '@quasar/extras/material-icons-round/material-icons-round.css'
import 'quasar/src/css/index.sass'

import App from './App.vue'
import router from './router'

const myApp = createApp(App)

myApp.use(router)

myApp.use(Quasar, {
  plugins: {
    Notify,
    Dialog,
  },
  lang: quasarLang,
  iconSet: quasarIconSet,
})

myApp.mount('#app')
