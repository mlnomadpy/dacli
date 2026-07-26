import { describe, it, expect } from 'vitest'
import { h } from 'vue'
import { mount } from '@vue/test-utils'
import { Button, buttonVariants } from '@/components/ui/button'
import { Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { Badge, badgeVariants } from '@/components/ui/badge'
import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'
import {
  Table,
  TableHeader,
  TableBody,
  TableRow,
  TableHead,
  TableCell,
} from '@/components/ui/table'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'

// Smoke coverage for the shadcn-vue foundation (task 151): each base component
// mounts, renders the expected element/slot, and threads a caller `class`
// through cn(). This is not a visual test — it guards the foundation against
// import/name/prop-shape regressions as sections get refactored onto it.

describe('shadcn-vue base components', () => {
  it('Button renders a <button> with variant classes and merges class', () => {
    const wrapper = mount(Button, {
      props: { variant: 'destructive', class: 'my-btn' },
      slots: { default: 'Retry' },
    })
    expect(wrapper.element.tagName).toBe('BUTTON')
    expect(wrapper.text()).toBe('Retry')
    expect(wrapper.attributes('class')).toContain('my-btn')
    expect(wrapper.attributes('class')).toContain('bg-destructive')
    expect(wrapper.attributes('data-slot')).toBe('button')
  })

  it('Button renders as a custom element via `as`', () => {
    const wrapper = mount(Button, { props: { as: 'a' }, slots: { default: 'link' } })
    expect(wrapper.element.tagName).toBe('A')
  })

  it('buttonVariants defaults to the primary variant', () => {
    expect(buttonVariants()).toContain('bg-primary')
  })

  it('Badge renders a <span> and honors the secondary variant', () => {
    const wrapper = mount(Badge, {
      props: { variant: 'secondary' },
      slots: { default: 'blocked' },
    })
    expect(wrapper.element.tagName).toBe('SPAN')
    expect(wrapper.text()).toBe('blocked')
    expect(wrapper.attributes('class')).toContain('bg-secondary')
    expect(badgeVariants()).toContain('bg-primary')
  })

  it('Card exposes the card surface and slots content', () => {
    const wrapper = mount(Card, { slots: { default: 'body' } })
    expect(wrapper.attributes('data-slot')).toBe('card')
    expect(wrapper.attributes('class')).toContain('bg-card')
    expect(wrapper.text()).toContain('body')
  })

  it('Card header/title/content render nested slots', () => {
    expect(mount(CardHeader).attributes('data-slot')).toBe('card-header')
    expect(mount(CardTitle, { slots: { default: 'core' } }).text()).toBe('core')
    expect(mount(CardContent).attributes('data-slot')).toBe('card-content')
  })

  it('Table renders a real <table> with head/body rows', () => {
    const wrapper = mount(Table, {
      slots: {
        default: `
          <thead data-slot="table-header"><tr><th>role</th></tr></thead>
          <tbody data-slot="table-body"><tr><td>root</td></tr></tbody>
        `,
      },
    })
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.find('[data-slot="table-container"]').exists()).toBe(true)
  })

  it('Table subcomponents render their semantic tags', () => {
    expect(mount(TableHeader).element.tagName).toBe('THEAD')
    expect(mount(TableBody).element.tagName).toBe('TBODY')
    expect(mount(TableRow).element.tagName).toBe('TR')
    expect(mount(TableHead).element.tagName).toBe('TH')
    expect(mount(TableCell).element.tagName).toBe('TD')
  })

  it('Progress reflects modelValue as an indicator transform', () => {
    const wrapper = mount(Progress, { props: { modelValue: 40 } })
    const indicator = wrapper.find('[data-slot="progress-indicator"]')
    expect(indicator.exists()).toBe(true)
    expect(indicator.attributes('style')).toContain('translateX(-60%)')
  })

  it('Separator is decorative and horizontal by default', () => {
    const wrapper = mount(Separator)
    expect(wrapper.attributes('data-slot')).toBe('separator')
    expect(wrapper.attributes('data-orientation')).toBe('horizontal')
  })

  it('Tooltip provides context so its trigger renders', () => {
    const wrapper = mount(Tooltip, {
      slots: {
        default: () => [h(TooltipTrigger, () => 'hover me'), h(TooltipContent, () => 'tip')],
      },
    })
    expect(wrapper.text()).toContain('hover me')
  })
})
