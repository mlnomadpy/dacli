// Tailwind + shadcn-vue theme tokens first; the hand-authored tokens.css (the
// legacy mission-control vars the not-yet-refactored sections still read) loads
// after and, being unlayered, keeps winning over Tailwind's base layer so the
// existing sections render unchanged.
import './assets/index.css'
import './assets/tokens.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
