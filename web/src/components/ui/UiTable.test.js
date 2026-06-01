import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import UiTable from './UiTable.vue'

describe('UiTable', () => {
  const columns = [
    { key: 'id', label: 'ID', sortable: true },
    { key: 'name', label: 'Name', sortable: true }
  ]
  const rows = [
    { id: 1, name: 'Alice' },
    { id: 2, name: 'Bob' }
  ]

  it('renders columns and rows', () => {
    const wrapper = mount(UiTable, {
      props: { columns, rows }
    })
    const ths = wrapper.findAll('th')
    expect(ths.length).toBe(2)
    expect(ths[0].text()).toContain('ID')

    const trs = wrapper.findAll('tbody tr')
    expect(trs.length).toBe(2)
    expect(trs[0].text()).toContain('Alice')
  })

  it('renders empty state when no rows', () => {
    const wrapper = mount(UiTable, {
      props: { columns, rows: [], emptyText: 'No users found' }
    })
    expect(wrapper.find('.ui-table__empty').exists()).toBe(true)
    expect(wrapper.find('.ui-table__empty').text()).toBe('No users found')
  })

  it('sorts rows when clicking sortable column header', async () => {
    const wrapper = mount(UiTable, {
      props: { columns, rows }
    })
    
    // Sort descending by name
    const nameTh = wrapper.findAll('th')[1]
    await nameTh.trigger('click') // Ascending
    await nameTh.trigger('click') // Descending

    const trs = wrapper.findAll('tbody tr')
    expect(trs[0].text()).toContain('Bob')
    expect(trs[1].text()).toContain('Alice')
  })

  it('paginates rows based on pageSize', () => {
    const wrapper = mount(UiTable, {
      props: { columns, rows, pageSize: 1 }
    })
    const trs = wrapper.findAll('tbody tr')
    expect(trs.length).toBe(1) // Only 1 row visible
    expect(wrapper.find('.ui-table__pagination').exists()).toBe(true)
  })

  it('emits row-click when a row is clicked', async () => {
    const wrapper = mount(UiTable, {
      props: { columns, rows }
    })
    const tr = wrapper.find('tbody tr')
    await tr.trigger('click')

    expect(wrapper.emitted('row-click')).toBeTruthy()
    expect(wrapper.emitted('row-click')[0][0]).toEqual(rows[0])
  })

  it('renders loading skeletons', () => {
    const wrapper = mount(UiTable, {
      props: { columns, loading: true }
    })
    expect(wrapper.find('.ui-table__skeleton').exists()).toBe(true)
  })
})
