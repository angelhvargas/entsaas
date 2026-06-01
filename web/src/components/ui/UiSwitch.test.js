import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiSwitch from '@/components/ui/UiSwitch.vue'

describe('UiSwitch', () => {
  it('renders the component', () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false } })
    expect(wrapper.find('.ui-switch').exists()).toBe(true)
  })

  it('renders label when provided', () => {
    const wrapper = mount(UiSwitch, {
      props: { modelValue: false, label: 'Enable notifications' },
    })
    expect(wrapper.find('.ui-switch__label').text()).toBe('Enable notifications')
  })

  it('track is off when modelValue=false', () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false } })
    expect(wrapper.find('.ui-switch__track').classes()).not.toContain('ui-switch__track--on')
  })

  it('track is on when modelValue=true', () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: true } })
    expect(wrapper.find('.ui-switch__track').classes()).toContain('ui-switch__track--on')
  })

  it('emits update:modelValue when clicked', async () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false } })
    await wrapper.find('button[role="switch"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })

  it('is disabled when disabled=true', () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false, disabled: true } })
    expect(wrapper.find('.ui-switch').classes()).toContain('ui-switch--disabled')
    expect(wrapper.find('button[role="switch"]').attributes('disabled')).toBeDefined()
  })

  it('applies sm size class', () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false, size: 'sm' } })
    expect(wrapper.find('.ui-switch').classes()).toContain('ui-switch--sm')
  })
})
