/**
 * Unit tests for UiButton component.
 */
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiButton from '@/components/ui/UiButton.vue'

describe('UiButton', () => {
  it('renders slot text', () => {
    const wrapper = mount(UiButton, { slots: { default: 'Click me' } })
    expect(wrapper.text()).toContain('Click me')
  })

  it('applies primary variant class by default', () => {
    const wrapper = mount(UiButton)
    expect(wrapper.classes()).toContain('ui-btn--primary')
  })

  it('applies danger variant class', () => {
    const wrapper = mount(UiButton, { props: { variant: 'danger' } })
    expect(wrapper.classes()).toContain('ui-btn--danger')
  })

  it('applies secondary variant class', () => {
    const wrapper = mount(UiButton, { props: { variant: 'secondary' } })
    expect(wrapper.classes()).toContain('ui-btn--secondary')
  })

  it('applies ghost variant class', () => {
    const wrapper = mount(UiButton, { props: { variant: 'ghost' } })
    expect(wrapper.classes()).toContain('ui-btn--ghost')
  })

  it('applies full-width class when full=true', () => {
    const wrapper = mount(UiButton, { props: { full: true } })
    expect(wrapper.classes()).toContain('ui-btn--full')
  })

  it('is disabled when loading=true', () => {
    const wrapper = mount(UiButton, { props: { loading: true } })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.classes()).toContain('ui-btn--loading')
  })

  it('is disabled when disabled=true', () => {
    const wrapper = mount(UiButton, { props: { disabled: true } })
    expect(wrapper.attributes('disabled')).toBeDefined()
  })

  it('emits click event when clicked', async () => {
    const wrapper = mount(UiButton)
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
  })

  it('renders with type=submit', () => {
    const wrapper = mount(UiButton, { props: { type: 'submit' } })
    expect(wrapper.attributes('type')).toBe('submit')
  })

  it('renders default type=button', () => {
    const wrapper = mount(UiButton)
    expect(wrapper.attributes('type')).toBe('button')
  })

  it('applies sm size class', () => {
    const wrapper = mount(UiButton, { props: { size: 'sm' } })
    expect(wrapper.classes()).toContain('ui-btn--sm')
  })

  it('applies lg size class', () => {
    const wrapper = mount(UiButton, { props: { size: 'lg' } })
    expect(wrapper.classes()).toContain('ui-btn--lg')
  })

  it('renders spinner when loading', () => {
    const wrapper = mount(UiButton, { props: { loading: true } })
    expect(wrapper.find('.ui-btn__spinner').exists()).toBe(true)
  })

  it('has no spinner when not loading', () => {
    const wrapper = mount(UiButton)
    expect(wrapper.find('.ui-btn__spinner').exists()).toBe(false)
  })
})
