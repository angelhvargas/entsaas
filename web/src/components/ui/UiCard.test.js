import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiCard from './UiCard.vue'

describe('UiCard', () => {
  it('renders default slot content', () => {
    const wrapper = mount(UiCard, {
      slots: {
        default: '<div class="test-body">Body Content</div>'
      }
    })
    expect(wrapper.find('.ui-card__body').exists()).toBe(true)
    expect(wrapper.find('.test-body').text()).toBe('Body Content')
  })

  it('renders header slot if provided', () => {
    const wrapper = mount(UiCard, {
      slots: {
        header: 'Card Header'
      }
    })
    expect(wrapper.find('.ui-card__header').exists()).toBe(true)
    expect(wrapper.find('.ui-card__header').text()).toBe('Card Header')
  })

  it('renders footer slot if provided', () => {
    const wrapper = mount(UiCard, {
      slots: {
        footer: '<button>Save</button>'
      }
    })
    expect(wrapper.find('.ui-card__footer').exists()).toBe(true)
    expect(wrapper.find('button').text()).toBe('Save')
  })

  it('applies padding classes correctly', () => {
    const wrapper = mount(UiCard, {
      props: { padding: 'lg' }
    })
    expect(wrapper.classes()).toContain('ui-card--pad-lg')
  })

  it('applies hover class if hover prop is true', () => {
    const wrapper = mount(UiCard, {
      props: { hover: true }
    })
    expect(wrapper.classes()).toContain('ui-card--hover')
  })

  it('removes border if border prop is false', () => {
    const wrapper = mount(UiCard, {
      props: { border: false }
    })
    expect(wrapper.classes()).toContain('ui-card--no-border')
  })
})
