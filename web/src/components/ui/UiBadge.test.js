import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiBadge from '@/components/ui/UiBadge.vue'

describe('UiBadge', () => {
  it('renders slot text', () => {
    const wrapper = mount(UiBadge, { slots: { default: 'Active' } })
    expect(wrapper.text()).toContain('Active')
  })

  it('applies default variant class', () => {
    const wrapper = mount(UiBadge, { slots: { default: '' } })
    expect(wrapper.find('.ui-badge').classes()).toContain('ui-badge--default')
  })

  it('applies success variant class', () => {
    const wrapper = mount(UiBadge, { props: { variant: 'success' }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge').classes()).toContain('ui-badge--success')
  })

  it('applies danger variant class', () => {
    const wrapper = mount(UiBadge, { props: { variant: 'danger' }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge').classes()).toContain('ui-badge--danger')
  })

  it('applies primary variant class', () => {
    const wrapper = mount(UiBadge, { props: { variant: 'primary' }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge').classes()).toContain('ui-badge--primary')
  })

  it('renders dot indicator when dot=true', () => {
    const wrapper = mount(UiBadge, { props: { dot: true }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge__dot').exists()).toBe(true)
  })

  it('does not render dot when dot=false', () => {
    const wrapper = mount(UiBadge, { props: { dot: false }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge__dot').exists()).toBe(false)
  })

  it('applies sm size class', () => {
    const wrapper = mount(UiBadge, { props: { size: 'sm' }, slots: { default: '' } })
    expect(wrapper.find('.ui-badge').classes()).toContain('ui-badge--sm')
  })
})
