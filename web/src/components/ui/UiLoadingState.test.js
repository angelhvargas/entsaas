import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiLoadingState from '@/components/ui/UiLoadingState.vue'

describe('UiLoadingState', () => {
  it('renders skeleton mode by default', () => {
    const wrapper = mount(UiLoadingState)
    expect(wrapper.find('.ui-loading__skeleton').exists()).toBe(true)
  })

  it('renders correct number of skeleton bones', () => {
    const wrapper = mount(UiLoadingState, { props: { mode: 'skeleton', lines: 4 } })
    expect(wrapper.findAll('.ui-loading__bone').length).toBe(4)
  })

  it('last bone is 60% width', () => {
    const wrapper = mount(UiLoadingState, { props: { mode: 'skeleton', lines: 3 } })
    const bones = wrapper.findAll('.ui-loading__bone')
    expect(bones[2].attributes('style')).toContain('width: 60%')
  })

  it('renders spinner mode', () => {
    const wrapper = mount(UiLoadingState, { props: { mode: 'spinner' } })
    expect(wrapper.find('.ui-loading__spinner-wrap').exists()).toBe(true)
    expect(wrapper.find('.ui-loading__spinner').exists()).toBe(true)
  })

  it('has aria-busy=true', () => {
    const wrapper = mount(UiLoadingState, { props: { mode: 'skeleton' } })
    expect(wrapper.find('[aria-busy="true"]').exists()).toBe(true)
  })

  it('spinner applies size class', () => {
    const wrapper = mount(UiLoadingState, { props: { mode: 'spinner', size: 'lg' } })
    expect(wrapper.find('.ui-loading__spinner').classes()).toContain('ui-loading__spinner--lg')
  })
})
