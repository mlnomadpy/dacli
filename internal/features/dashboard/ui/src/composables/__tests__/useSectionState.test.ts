import { describe, it, expect } from 'vitest'
import { sectionState } from '../useSectionState'

describe('sectionState', () => {
  it('is loading before the first snapshot regardless of slice', () => {
    expect(sectionState('loading', false, true)).toBe('loading')
    expect(sectionState('loading', false, false)).toBe('loading')
  })

  it('shows a cold error only when a poll failed with no prior snapshot', () => {
    expect(sectionState('error', false, true)).toBe('error')
    expect(sectionState('error', false, false)).toBe('error')
  })

  it('keeps a retained snapshot live/empty on a later poll failure', () => {
    // hasSnapshot === true → the header owns the error; the section stays on data.
    expect(sectionState('error', true, false)).toBe('live')
    expect(sectionState('error', true, true)).toBe('empty')
  })

  it('resolves empty vs live from the slice once live', () => {
    expect(sectionState('live', true, true)).toBe('empty')
    expect(sectionState('live', true, false)).toBe('live')
  })
})
