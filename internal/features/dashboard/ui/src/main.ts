// Tailwind + shadcn-vue theme tokens first (the sections now build entirely on
// these); then tokens.css, trimmed by task 152 to only the residual scaffolding
// the theme doesn't express — the four status hues, the live-pulse keyframe, the
// `.mono` helper, and the body reset. It no longer shadows the shadcn palette.
import './assets/index.css'
import './assets/tokens.css'

import { createApp } from 'vue'
import { createPinia } from 'pinia'

import App from './App.vue'

const app = createApp(App)
app.use(createPinia())
app.mount('#app')
