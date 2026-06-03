<template>
  <div v-if="open" class="modal-backdrop" @click.self="$emit('close')"><section class="sheet"><div class="sheet-head"><div><h2>申请加入流动摊主</h2><p class="muted">提交后平台会联系你确认资质。</p></div><button class="c-btn ghost" @click="$emit('close')">关闭</button></div><div class="form-grid"><label class="field">店铺名<input v-model="form.shopName" class="c-input"></label><label class="field">摊位类型<select v-model="form.category" class="c-select"><option v-for="item in categories" :key="item">{{ item }}</option></select></label><label class="field">联系人<input v-model="form.contactName" class="c-input"></label><label class="field">手机号<input v-model="form.contactPhone" class="c-input" inputmode="tel" placeholder="请输入手机号"></label><label class="field full">常出摊区域<input v-model="form.usualArea" class="c-input"></label><label class="field full">补充说明<input v-model="form.remark" class="c-input"></label><label class="field full">流动摊照片<input class="c-input" type="file" accept="image/*" @change="readImage"></label></div><div v-if="result" class="result">{{ result }}</div><div v-if="error" class="result error">{{ error }}</div><div class="card-actions"><button class="c-btn secondary" @click="$emit('close')">稍后再说</button><button class="c-btn primary" :disabled="submitting" @click="submit">{{ submitting ? '提交中…' : '提交申请' }}</button></div></section></div>
</template>
<script setup>
import { reactive, ref, watch } from 'vue'
import { apiFetch } from '../../../api/client'
import { STALL_CATEGORY_OPTIONS } from '../categoryIcons'
const props = defineProps({ open: Boolean, storedContact: Object })
const emit = defineEmits(['close', 'save-contact'])
const categories = STALL_CATEGORY_OPTIONS
const form = reactive({ shopName: '', category: '早餐小吃', contactName: '', contactPhone: '', usualArea: '', remark: '', photoUrl: '' })
const error = ref(''); const result = ref(''); const submitting = ref(false)
watch(() => props.open, (open) => { if (open) { form.contactName ||= props.storedContact?.name || ''; form.contactPhone ||= props.storedContact?.phone || ''; error.value = ''; result.value = '' } })
function validPhone(phone) { const d = String(phone || '').replace(/\D/g, ''); return (d.length === 11 && d.startsWith('1')) || d.length === 8 || (d.length === 11 && d.startsWith('852')) }
function readImage(e) { const file = [...(e.target.files || [])].find((item) => item.type.startsWith('image/')); if (!file) return; const reader = new FileReader(); reader.onload = () => { form.photoUrl = String(reader.result || '') }; reader.readAsDataURL(file) }
async function submit() { if (!form.shopName || !form.contactName || !form.contactPhone || !form.photoUrl) { error.value = '请填写必填信息并上传摊位照片'; return } if (!validPhone(form.contactPhone)) { error.value = '请输入有效手机号'; return } submitting.value = true; error.value = ''; result.value = ''; try { const resp = await apiFetch('/api/merchant-applications', { method: 'POST', body: JSON.stringify({ shop_name: form.shopName, contact_name: form.contactName, contact_phone: form.contactPhone, category: form.category, photo_url: form.photoUrl, usual_area: form.usualArea, remark: form.remark }) }); emit('save-contact', { name: form.contactName, phone: form.contactPhone }); result.value = `申请已提交，编号 #${resp.application?.id || '待生成'}。平台会尽快联系你确认。` } catch (e) { error.value = e.message || '提交失败' } finally { submitting.value = false } }
</script>
