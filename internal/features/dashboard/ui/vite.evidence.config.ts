import { fileURLToPath, URL } from 'node:url'

import tailwindcss from '@tailwindcss/vite'
import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'
import { viteSingleFile } from 'vite-plugin-singlefile'

// Build a self-contained, read-only representative workspace for manual
// browser verification and documentation captures. This is never embedded in
// dacli or served by the dashboard command; production continues to read only
// the Go API. Keeping the harness separate prevents a query string or browser
// flag from replacing real workspace evidence in a released binary.
export default defineConfig({
  plugins: [vue(), tailwindcss(), viteSingleFile()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    target: 'es2022',
    outDir: 'evidence-dist',
    emptyOutDir: true,
    assetsInlineLimit: 100_000_000,
    cssCodeSplit: false,
    reportCompressedSize: false,
    rollupOptions: {
      input: fileURLToPath(new URL('./evidence.html', import.meta.url)),
    },
  },
})
