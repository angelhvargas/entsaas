import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { markRaw } from 'vue'
import UiInput from '@/components/ui/UiInput.vue'

describe('UiInput', () => {
  it('renders an input element', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '' } })
    expect(wrapper.find('input').exists()).toBe(true)
  })

  it('binds modelValue to the input value', () => {
    const wrapper = mount(UiInput, { props: { modelValue: 'hello' } })
    expect(wrapper.find('input').element.value).toBe('hello')
  })

  it('emits update:modelValue on input', async () => {
    const wrapper = mount(UiInput, { props: { modelValue: '' } })
    await wrapper.find('input').setValue('world')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['world'])
  })

  it('renders placeholder', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '', placeholder: 'Enter email' } })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Enter email')
  })

  it('applies error class when error prop is set', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '', error: 'Required' } })
    expect(wrapper.find('.ui-input').classes()).toContain('ui-input--error')
  })

  it('does not show error class when no error', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '' } })
    expect(wrapper.find('.ui-input').classes()).not.toContain('ui-input--error')
  })

  it('renders as type=password', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '', type: 'password' } })
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  it('renders left icon slot when icon provided', () => {
    const MockIcon = markRaw({ template: '<svg data-testid="icon" />' })
    const wrapper = mount(UiInput, { props: { modelValue: '', icon: MockIcon } })
    expect(wrapper.find('.ui-input__wrap--icon-left').exists()).toBe(true)
  })

  it('is disabled when disabled=true', () => {
    const wrapper = mount(UiInput, { props: { modelValue: '', disabled: true } })
    expect(wrapper.find('input').attributes('disabled')).toBeDefined()
  })
})
