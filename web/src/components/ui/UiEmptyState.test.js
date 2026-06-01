import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiEmptyState from './UiEmptyState.vue'

import { markRaw } from 'vue'

// A dummy icon component for testing
const DummyIcon = markRaw({
  template: '<svg class="dummy-icon" />'
})

describe('UiEmptyState', () => {
  it('renders default title', () => {
    const wrapper = mount(UiEmptyState)
    expect(wrapper.find('.ui-empty__title').text()).toBe('Nothing here')
  })

  it('renders custom title and description', () => {
    const wrapper = mount(UiEmptyState, {
      props: {
        title: 'Custom Title',
        description: 'Custom Description'
      }
    })
    expect(wrapper.find('.ui-empty__title').text()).toBe('Custom Title')
    expect(wrapper.find('.ui-empty__desc').text()).toBe('Custom Description')
  })

  it('renders icon prop', () => {
    const wrapper = mount(UiEmptyState, {
      props: {
        icon: DummyIcon
      }
    })
    expect(wrapper.find('.dummy-icon').exists()).toBe(true)
  })

  it('renders icon slot', () => {
    const wrapper = mount(UiEmptyState, {
      slots: {
        icon: '<div class="custom-icon-slot">Icon</div>'
      }
    })
    expect(wrapper.find('.custom-icon-slot').exists()).toBe(true)
  })

  it('renders action button and emits click', async () => {
    const wrapper = mount(UiEmptyState, {
      props: {
        actionLabel: 'Click Me',
        actionVariant: 'danger'
      }
    })
    const btn = wrapper.find('button')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toBe('Click Me')
    expect(btn.classes()).toContain('btn-danger')

    await btn.trigger('click')
    expect(wrapper.emitted('action')).toBeTruthy()
  })

  it('renders default slot over action button', () => {
    const wrapper = mount(UiEmptyState, {
      props: {
        actionLabel: 'Click Me'
      },
      slots: {
        default: '<div class="custom-action">Custom Action</div>'
      }
    })
    expect(wrapper.find('button').exists()).toBe(false)
    expect(wrapper.find('.custom-action').exists()).toBe(true)
  })
})
