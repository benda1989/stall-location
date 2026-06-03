<template>
  <div class="m-pull-search-zone">
    <div v-if="open" class="m-pull-search-panel">
      <div class="m-search-input-wrap">
        <input ref="inputEl" v-model="draft" class="m-input" type="search" :placeholder="placeholder" @keyup.enter="submit" @search="handleNativeSearch">
        <button v-if="draft" class="m-search-clear" type="button" aria-label="清空检索" @click="clearInput">×</button>
      </div>
    </div>
    <button v-else type="button" class="m-pull-search-hint" :class="{ 'is-visible': pulling || modelValue }" @click="reveal">
      {{ modelValue ? `当前检索：${modelValue}` : pulling ? '松手打开检索' : '到顶后继续下拉可检索' }}
    </button>
  </div>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({ modelValue: String, placeholder: String })
const emit = defineEmits(['update:modelValue', 'search', 'clear'])
const open = ref(false)
const pulling = ref(false)
const inputEl = ref(null)
const draft = ref(props.modelValue || '')
let startY = 0
let activePull = false

watch(() => props.modelValue, (value) => { draft.value = value || '' })

function atTop() {
  return window.scrollY <= 4 || document.documentElement.scrollTop <= 4 || document.body.scrollTop <= 4
}
function reveal() {
  if (open.value) return
  draft.value = props.modelValue || ''
  open.value = true
  pulling.value = false
  activePull = false
  nextTick(() => inputEl.value?.focus())
}
function onTouchStart(event) {
  if (open.value || !atTop()) return
  startY = event.touches?.[0]?.clientY || 0
  activePull = true
  pulling.value = false
}
function onTouchMove(event) {
  if (!activePull || !atTop() || open.value) return
  const currentY = event.touches?.[0]?.clientY || 0
  const distance = currentY - startY
  pulling.value = distance > 18
  if (distance > 56) reveal()
}
function onTouchEnd() {
  activePull = false
  pulling.value = false
}
function onWheel(event) {
  if (atTop() && event.deltaY < -36) reveal()
}
function submit() {
  emit('update:modelValue', draft.value.trim())
  open.value = false
  pulling.value = false
  inputEl.value?.blur?.()
  emit('search')
}
function clearInput() {
  draft.value = ''
  emit('update:modelValue', '')
  emit('clear')
  nextTick(() => inputEl.value?.focus())
}
function handleNativeSearch(event) {
  if (!event.target.value) clearInput()
}

onMounted(() => {
  window.addEventListener('touchstart', onTouchStart, { passive: true })
  window.addEventListener('touchmove', onTouchMove, { passive: true })
  window.addEventListener('touchend', onTouchEnd, { passive: true })
  window.addEventListener('wheel', onWheel, { passive: true })
})
onBeforeUnmount(() => {
  window.removeEventListener('touchstart', onTouchStart)
  window.removeEventListener('touchmove', onTouchMove)
  window.removeEventListener('touchend', onTouchEnd)
  window.removeEventListener('wheel', onWheel)
})
</script>
