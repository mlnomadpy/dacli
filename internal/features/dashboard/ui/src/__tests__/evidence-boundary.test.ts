import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'

import { describe, expect, it } from 'vitest'

function source(relative: string): string {
  return readFileSync(fileURLToPath(new URL(relative, import.meta.url)), 'utf8')
}

describe('representative browser evidence boundary', () => {
  it('keeps fixture data out of the released dashboard entry', () => {
    const productionEntry = source('../main.ts')
    const productionConfig = source('../../vite.config.ts')
    const evidenceEntry = source('../evidence.ts')
    const evidenceConfig = source('../../vite.evidence.config.ts')

    expect(productionEntry).not.toContain('dashboard-state.json')
    expect(productionEntry).not.toContain('fixtureResponse')
    expect(productionConfig).not.toContain('evidence.html')
    expect(evidenceEntry).toContain('dashboard-state.json')
    expect(evidenceEntry).toContain('window.fetch =')
    expect(evidenceConfig).toContain("input: fileURLToPath(new URL('./evidence.html'")
    expect(evidenceConfig).toContain("outDir: 'evidence-dist'")
  })

  it('keeps analytics and attention evidence visible in the shared fixture entry', () => {
    const evidenceEntry = source('../evidence.ts')

    expect(evidenceEntry).toContain("schema: 'operator-attention/v1'")
    expect(evidenceEntry).toContain("schema: 'outcome-analytics/v1'")
    expect(evidenceEntry).toContain('Missing cost remains unknown')
    expect(evidenceEntry).toContain('absence is not treated as healthy')
    expect(evidenceEntry).toContain("evidenceMode === 'cold-error'")
    expect(evidenceEntry).toContain("evidenceMode === 'stale-outcomes'")
    expect(evidenceEntry).toContain("window.location.hash = '#/overview?project=dacli'")
    expect(evidenceEntry).not.toContain('live=paused')
  })
})
