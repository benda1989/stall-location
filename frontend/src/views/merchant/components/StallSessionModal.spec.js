import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import StallSessionModal from './StallSessionModal.vue'
import { getLocation } from '../../../api/wechat'

vi.mock('../../../api/wechat', () => ({
  getLocation: vi.fn()
}))

function flushPromises() {
  return new Promise((resolve) => setTimeout(resolve, 0))
}

describe('StallSessionModal', () => {
  beforeEach(() => {
    vi.mocked(getLocation).mockReset()
    const storage = new Map()
    vi.stubGlobal('localStorage', {
      getItem: (key) => storage.get(key) || null,
      setItem: (key, value) => storage.set(key, String(value)),
      removeItem: (key) => storage.delete(key),
      clear: () => storage.clear()
    })
    window.history.replaceState({}, '', '/merchant/dashboard?preview=1')
  })

  it('falls back to a development location when browser geolocation is denied', async () => {
    vi.mocked(getLocation).mockRejectedValue(new Error('User denied Geolocation'))
    const wrapper = mount(StallSessionModal, { props: { open: false } })

    await wrapper.setProps({ open: true })
    await flushPromises()
    await nextTick()

    expect(wrapper.text()).toContain('已切换到开发定位：旺角 E2 口')
    expect(wrapper.find('input[placeholder="例如 小区南门便利店旁"]').element.value).toBe('旺角地铁站 E2 口附近')

    const startButton = wrapper.findAll('button').find((button) => button.text() === '开始出摊')
    expect(startButton.exists()).toBe(true)
    expect(startButton.element.disabled).toBe(false)

    await startButton.trigger('click')
    const submit = wrapper.emitted('submit')?.[0]?.[0]
    expect(submit).toMatchObject({
      lat: 22.3193,
      lng: 114.1694,
      address: '旺角地铁站 E2 口附近',
      location_accuracy: 12
    })
  })
})
