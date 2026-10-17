import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { autoAnimatePlugin } from '@formkit/auto-animate/vue'
import { piniaPersistPlugin } from './plugins/piniaPersist'
import './style.css'
import App from './App.vue'

const app = createApp(App)
const pinia = createPinia()
pinia.use(piniaPersistPlugin)

app.use(pinia)
app.use(autoAnimatePlugin)
app.mount('#app')