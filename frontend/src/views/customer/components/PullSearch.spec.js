import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

import PullSearch from './PullSearch.vue'

function mountPullSearch(props = {}) {
  return mount(PullSearch, {
    props: {
      modelValue: '',
      placeholder: '检索',
      ...props
    }
  })
}

describe('PullSearch', () => {
  it('opens and collapses after submit', async () => {
    const wrapper = mountPullSearch()

    expect(wrapper.find('.pull-search-panel').exists()).toBe(false)

    await wrapper.find('.pull-search-hint').trigger('click')
    expect(wrapper.find('.pull-search-panel').exists()).toBe(true)

    const input = wrapper.find('input.c-input')
    await input.setValue('  咖啡 ')
    await input.trigger('keyup.enter')
    await nextTick()

    expect(wrapper.find('.pull-search-panel').exists()).toBe(false)
    expect(wrapper.emitted('update:modelValue')).toEqual([['咖啡']])
    expect(wrapper.emitted('search')).toHaveLength(1)
  })

  it('clears draft and emits clear event', async () => {
    const wrapper = mountPullSearch({ modelValue: '煎饼' })

    await wrapper.find('.pull-search-hint').trigger('click')
    const clear = wrapper.find('button.search-clear')
    expect(clear.exists()).toBe(true)

    await clear.trigger('click')
    await nextTick()

    expect(wrapper.emitted('update:modelValue')).toEqual([['']])
    expect(wrapper.emitted('clear')).toHaveLength(1)
    expect(wrapper.find('input.c-input').element.value).toBe('')
  })

  it('reveals when user wheels up at top', async () => {
    const wrapper = mountPullSearch()

    window.dispatchEvent(new WheelEvent('wheel', { deltaY: -60 }))
    await nextTick()

    expect(wrapper.find('.pull-search-panel').exists()).toBe(true)
  })
})
