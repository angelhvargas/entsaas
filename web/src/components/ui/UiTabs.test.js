import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiTabs from './UiTabs.vue'

describe('UiTabs', () => {
  const tabs = [
    { key: 'general', label: 'General' },
    { key: 'billing', label: 'Billing' },
    { key: 'advanced', label: 'Advanced' }
  ]

  it('renders a list of tabs', () => {
    const wrapper = mount(UiTabs, {
      props: { tabs }
    })
    const buttons = wrapper.findAll('.ui-tabs__tab')
    expect(buttons.length).toBe(3)
    expect(buttons[0].text()).toContain('General')
    expect(buttons[1].text()).toContain('Billing')
  })

  it('defaults to first tab if modelValue is empty', () => {
    const wrapper = mount(UiTabs, {
      props: { tabs }
    })
    const buttons = wrapper.findAll('.ui-tabs__tab')
    expect(buttons[0].classes()).toContain('ui-tabs__tab--active')
  })

  it('respects modelValue prop', () => {
    const wrapper = mount(UiTabs, {
      props: { tabs, modelValue: 'billing' }
    })
    const buttons = wrapper.findAll('.ui-tabs__tab')
    expect(buttons[1].classes()).toContain('ui-tabs__tab--active')
  })

  it('emits update:modelValue when a tab is clicked', async () => {
    const wrapper = mount(UiTabs, {
      props: { tabs, modelValue: 'general' }
    })
    const buttons = wrapper.findAll('.ui-tabs__tab')
    await buttons[2].trigger('click') // Click Advanced
    
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')[0][0]).toBe('advanced')
  })

  it('updates active tab when modelValue changes', async () => {
    const wrapper = mount(UiTabs, {
      props: { tabs, modelValue: 'general' }
    })
    await wrapper.setProps({ modelValue: 'billing' })
    const buttons = wrapper.findAll('.ui-tabs__tab')
    expect(buttons[1].classes()).toContain('ui-tabs__tab--active')
  })
})
