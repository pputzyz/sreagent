import './utils/dataview-polyfill'
import '@fontsource/inter/400.css'
import '@fontsource/inter/500.css'
import '@fontsource/inter/600.css'
import '@fontsource/inter/700.css'
import '@fontsource/jetbrains-mono/400.css'
import '@fontsource/jetbrains-mono/500.css'
import '@fontsource/jetbrains-mono/600.css'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import i18n from './i18n'
import router from './router'
import { vRipple } from '@/directives/ripple'
import './styles/global.css'
import 'vue-virtual-scroller/dist/vue-virtual-scroller.css'

const app = createApp(App)

app.use(createPinia())
app.use(i18n)
app.use(router)

app.directive('ripple', vRipple)

app.mount('#app')
