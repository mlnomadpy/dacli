import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BoardColumn from '../BoardColumn.vue'

describe('BoardColumn', () => {
  it('renders the true count and one chip per task up to the count', () => {
    const w = mount(BoardColumn, { props: { status: 'open', count: 5 } })
    expect(w.find('.count').text()).toBe('5')
    expect(w.findAll('.chip')).toHaveLength(5)
    expect(w.find('.more').exists()).toBe(false)
  })

  it('caps chips at 24 and shows "+K more" while the header keeps the true total', () => {
    const w = mount(BoardColumn, { props: { status: 'done', count: 500 } })
    expect(w.find('.count').text()).toBe('500') // header is always the true total
    expect(w.findAll('.chip')).toHaveLength(24) // chips capped
    expect(w.find('.more').text()).toBe('+476 more')
  })

  it('renders an em-dash placeholder, not a blank gap, for a zero column', () => {
    const w = mount(BoardColumn, { props: { status: 'blocked', count: 0 } })
    expect(w.find('.count').text()).toBe('0')
    expect(w.findAll('.chip')).toHaveLength(0)
    expect(w.find('.none').text()).toBe('—')
  })

  it('labels the column region for screen readers', () => {
    const w = mount(BoardColumn, { props: { status: 'active', count: 3 } })
    expect(w.find('[role="group"]').attributes('aria-label')).toBe('active — 3 tasks')
  })
})
