<template>
  <div v-if="open" class="m-modal" @click.self="$emit('close')">
    <section class="m-sheet">
      <div class="m-sheet-head">
        <div>
          <h2>开始出摊</h2>
          <p class="muted">系统会自动定位，定位成功后才能公开摊位。</p>
        </div>
        <button class="m-btn secondary" @click="$emit('close')">关闭</button>
      </div>

      <div class="m-form">
        <label class="m-field">预计结束<input v-model="form.expected_end_at" class="m-input" type="datetime-local"></label>
        <label class="m-field">位置描述<input v-model="form.address" class="m-input" placeholder="例如 小区南门便利店旁"></label>
        <label v-if="devLocationEnabled" class="m-field full dev-location-field">
          开发定位
          <select class="m-select" @change="chooseDevLocation">
            <option value="">选择一个测试点位，替代浏览器/微信定位</option>
            <option v-for="location in devLocations" :key="location.name" :value="location.name">{{ location.name }} · {{ location.address }}</option>
          </select>
        </label>
        <label class="m-field full">出摊照片<input type="file" class="m-input" accept="image/*" capture="environment" @change="readPhoto"></label>
      </div>

      <div class="m-result" :class="{ error: locationError }">{{ locationText || '打开弹窗后会自动定位，请允许浏览器/微信获取位置。' }}</div>
      <div v-if="form.photo_url" class="stall-photo-preview"><img :src="form.photo_url" alt="今日出摊照片预览"></div>
      <div class="m-actions">
        <button class="m-btn secondary" type="button" :disabled="locating" @click="locate">{{ locating ? '定位中…' : '重新定位' }}</button>
        <button class="m-btn secondary" @click="$emit('close')">取消</button>
        <button class="m-btn primary" :disabled="!canStart" @click="$emit('submit', payload())">{{ canStart ? '开始出摊' : '定位成功后可出摊' }}</button>
      </div>
    </section>
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { getLocation } from '../../../api/wechat'

const props = defineProps({ open: Boolean })
defineEmits(['close', 'submit'])
const devLocationEnabled = computed(() => import.meta.env.DEV || new URLSearchParams(window.location.search).get('preview') === '1')
const devLocations = [
  { name: '旺角 E2 口', lat: 22.3193, lng: 114.1694, accuracy: 12, address: '旺角地铁站 E2 口附近' },
  { name: '创意园东门', lat: 22.3221, lng: 114.1712, accuracy: 18, address: '创意园东门树荫下' },
  { name: '社区南门', lat: 22.3168, lng: 114.1678, accuracy: 15, address: '社区南门便利店旁' },
  { name: '青榕广场', lat: 22.3242, lng: 114.1669, accuracy: 20, address: '青榕广场西南角' },
  { name: '海棠社区东门', lat: 22.3176, lng: 114.1751, accuracy: 16, address: '海棠社区东门便民点' }
]
const form = reactive({ expected_end_at: '', address: '', lat: 0, lng: 0, accuracy: 0, photo_url: '' })
const locating = ref(false)
const hasLocated = ref(false)
const locationText = ref('')
const locationError = ref(false)
const canStart = computed(() => hasLocated.value && !locating.value && form.expected_end_at && form.address)

watch(() => props.open, (open) => {
  if (!open) return
  form.expected_end_at = localDateTime(new Date(Date.now() + 4 * 3600 * 1000))
  form.address = ''
  form.lat = 0
  form.lng = 0
  form.accuracy = 0
  form.photo_url = ''
  hasLocated.value = false
  locationText.value = ''
  locationError.value = false
  locate()
})

function localDateTime(date) {
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

async function locate() {
  locating.value = true
  locationText.value = '正在自动定位，请保持页面打开…'
  locationError.value = false
  hasLocated.value = false
  try {
    const location = await getLocation()
    form.lat = Number(location.lat)
    form.lng = Number(location.lng)
    form.accuracy = Number(location.accuracy || 0)
    form.address ||= `自动定位点 ${form.lat.toFixed(5)}, ${form.lng.toFixed(5)}`
    hasLocated.value = true
    locationText.value = `已自动定位，精度约 ${form.accuracy || '-'} 米。可补充位置描述后出摊。`
  } catch (error) {
    if (devLocationEnabled.value) {
      applyDevLocation(devLocations[0])
      locationText.value = `${error.message || '自动定位失败'}，已切换到开发定位：${devLocations[0].name}。`
      return
    }
    locationError.value = true
    locationText.value = error.message || '自动定位失败，无法出摊；请允许定位后重试。'
  } finally {
    locating.value = false
  }
}

function chooseDevLocation(event) {
  const location = devLocations.find((item) => item.name === event.target.value)
  if (!location) return
  event.target.value = ''
  applyDevLocation(location)
  locationText.value = `已使用开发定位：${location.name}，精度约 ${form.accuracy} 米。`
}

function applyDevLocation(location) {
  localStorage.setItem('mplzDevLocation', JSON.stringify(location))
  form.lat = Number(location.lat)
  form.lng = Number(location.lng)
  form.accuracy = Number(location.accuracy || 0)
  form.address = location.address
  hasLocated.value = true
  locationError.value = false
}

function readPhoto(event) {
  const file = [...(event.target.files || [])].find((item) => item.type.startsWith('image/'))
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => { form.photo_url = String(reader.result || '') }
  reader.readAsDataURL(file)
}

function payload() {
  return { lat: form.lat, lng: form.lng, address: form.address, expected_end_at: new Date(form.expected_end_at).toISOString(), location_accuracy: Math.round(form.accuracy || 0), photo_url: form.photo_url }
}
</script>
