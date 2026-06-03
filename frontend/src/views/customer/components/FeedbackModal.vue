<template>
  <div v-if="open" class="modal-backdrop" @click.self="$emit('close')"><section class="sheet"><div class="sheet-head"><div><h2>问题反馈</h2><p class="muted">描述问题、上传图片并留下手机号。</p></div><button class="c-btn ghost" @click="$emit('close')">关闭</button></div><div class="form-grid"><label class="field full">反馈描述<textarea v-model="form.description" class="c-textarea"></textarea></label><label class="field">联系人<input v-model="form.contactName" class="c-input"></label><label class="field">手机号<input v-model="form.contactPhone" class="c-input" inputmode="tel" placeholder="请输入手机号"></label><label class="field full">图片<input class="c-input" type="file" accept="image/*" @change="readImage"></label></div><div v-if="result" class="result">{{ result }}</div><div v-if="error" class="result error">{{ error }}</div><div class="card-actions"><button class="c-btn secondary" @click="$emit('close')">取消</button><button class="c-btn primary" :disabled="submitting" @click="submit">{{ submitting ? '提交中…' : '提交反馈' }}</button></div></section></div>
</template>
<script setup>
import { reactive, ref, watch } from 'vue'
import { apiFetch } from '../../../api/client'
const props = defineProps({ open: Boolean, storedContact: Object })
const emit = defineEmits(['close', 'save-contact'])
const form = reactive({ description: '', contactName: '', contactPhone: '', imageUrl: '' })
const error = ref(''); const result = ref(''); const submitting = ref(false)
watch(() => props.open, (open) => { if (open) { form.contactName ||= props.storedContact?.name || ''; form.contactPhone ||= props.storedContact?.phone || ''; error.value = ''; result.value = '' } })
function validPhone(phone) { const d = String(phone || '').replace(/\D/g, ''); return (d.length === 11 && d.startsWith('1')) || d.length === 8 || (d.length === 11 && d.startsWith('852')) }
function readImage(e) { const file = [...(e.target.files || [])].find((item) => item.type.startsWith('image/')); if (!file) return; const reader = new FileReader(); reader.onload = () => { form.imageUrl = String(reader.result || '') }; reader.readAsDataURL(file) }
async function submit() { if (!form.description || !form.contactPhone) { error.value = '请填写描述和手机号'; return } if (!validPhone(form.contactPhone)) { error.value = '请输入有效手机号'; return } submitting.value = true; error.value = ''; result.value = ''; try { await apiFetch('/api/feedback', { method: 'POST', body: JSON.stringify({ source: 'customer', contact_name: form.contactName, contact_phone: form.contactPhone, description: form.description, image_url: form.imageUrl, page_url: location.pathname + location.search }) }); emit('save-contact', { name: form.contactName, phone: form.contactPhone }); result.value = '反馈已提交，平台会尽快处理。'; form.description = ''; form.imageUrl = '' } catch (e) { error.value = e.message || '提交失败' } finally { submitting.value = false } }
</script>
