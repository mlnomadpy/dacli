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
    // Do NOT wipe dist/ on build. The committed dist/.gitkeep keeps the
    // directory tracked so `//go:embed all:ui/dist` (../dashboard.go) always has
    // a target; emptying dist would delete that tracked placeholder and dirty
    // the working tree — which would break `goreleaser release` (it refuses a
    // dirty repo). viteSingleFile emits only index.html, so nothing stale
    // accumulates.
    emptyOutDir: false,
  },
  // Dev mode: `npm run dev` serves the SPA on Vite's own port with HMR and
  // proxies the API to a running `dacli dashboard --port 8787`, so the live
  // Go endpoints back the hot-reloading UI. Run the two side by side:
  //   dacli dashboard --port 8787   # terminal 1 (the Go API + embedded page)
  //   npm run dev                   # terminal 2 (this Vite server, proxied)
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8787',
    },
  },
})
