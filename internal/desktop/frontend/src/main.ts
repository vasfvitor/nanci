import { createApp } from 'vue'
import { Quasar, Notify, Dialog } from 'quasar'
import quasarLang from 'quasar/lang/pt-BR'
import quasarIconSet from 'quasar/icon-set/material-icons'

import '@fontsource/inter/index.css'
import '@quasar/extras/material-icons/material-icons.css'
import 'quasar/src/css/index.sass'
import './css/app.scss'

import App from './App.vue'
import router from './router'
import { createPinia } from 'pinia'
import { useConsoleStore } from '@/stores/console'
import { desktopRuntime } from '@/platform/wails/runtime'
import { errorMessage } from '@/platform/wails/client'
import { readStorage } from '@/utils/storage'

const myApp = createApp(App)
const pinia = createPinia()

myApp.use(pinia)
myApp.use(router)

const savedDark = readStorage('darkMode')
let initialDark: boolean | 'auto' = 'auto'
if (savedDark === 'true') {
  initialDark = true
} else if (savedDark === 'false') {
  initialDark = false
}
desktopRuntime.setWindowTheme(initialDark)

myApp.use(Quasar, {
  plugins: {
    Notify,
    Dialog,
  },
  config: {
    dark: initialDark,
  },
  lang: quasarLang,
  iconSet: quasarIconSet,
})

const consoleStore = useConsoleStore(pinia)
consoleStore.initLogListeners()
consoleStore.syncInitialLogLevel().catch((error: unknown) => {
  Notify.create({
    type: 'negative',
    message: 'Não foi possível aplicar o nível de log inicial: ' + errorMessage(error),
  })
})

myApp.mount('#app')
