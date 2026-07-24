import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { viteSingleFile } from 'vite-plugin-singlefile'

// The dashboard ships as ONE self-contained file that Go embeds exactly the way
// it embeds `static/index.html` today: no CDN, no runtime network fetch except
// `/api/state` (DESIGN.md §2). `viteSingleFile` inlines the JS/CSS into the HTML
// so the built `dist/index.html` is the whole artifact.
// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue(), viteSingleFile()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    target: 'es2022',
    // Everything inlined; assetsInlineLimit high so nothing is emitted alongside.
    assetsInlineLimit: 100_000_000,
    cssCodeSplit: false,
    reportCompressedSize: false,
  },
})
