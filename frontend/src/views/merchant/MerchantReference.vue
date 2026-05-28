<template>
  <main v-if="!token" class="page merchant-flow" aria-label="摊主登录">
    <section class="hero merchant-flow-hero">
      <div>
        <div class="eyebrow">Merchant Login</div>
        <h1>手机号验证码登录</h1>
        <p>摊主登录后会按申请/审核结果自动分流：无店铺先申请，处理中看结果，通过后进入工作台。</p>
      </div>
    </section>
    <section class="card merchant-auth-card form-stack">
      <van-field v-model="auth.phone" label="手机号" placeholder="请输入手机号" />
      <div class="sms-row">
        <van-field v-model="auth.code" label="验证码" placeholder="本地可用 123456" />
        <van-button round plain @click="sendCode">发送验证码</van-button>
      </div>
      <van-button block round color="#60A5FA" :loading="submitting" @click="login">登录并查看状态</van-button>
    </section>
  </main>

  <main v-else-if="!showWorkbench" class="page merchant-flow" aria-label="摊主申请状态">
    <section class="hero merchant-flow-hero">
      <div>
        <div class="eyebrow">Application Status</div>
        <h1>{{ statusTitle }}</h1>
        <p>{{ statusDescription }}</p>
      </div>
      <div class="merchant-status-chip">
        <span class="pill" :class="statusPillClass">{{ application?.status || status?.next_action || 'apply' }}</span>
        <strong>{{ application?.application_no || application?.id ? `#${application.application_no || application.id}` : '待提交' }}</strong>
      </div>
    </section>

    <section v-if="application && application.review_reason" class="card result-card">
      <span class="pill red">处理说明</span>
      <p>{{ application.review_reason }}</p>
    </section>

    <section v-if="needsApplicationForm" class="card form-stack merchant-apply-card">
      <div class="between">
        <div>
          <span class="pill green">Stall Apply</span>
          <h2>{{ application?.id ? '修改并重新提交' : '申请加入流动摊主' }}</h2>
        </div>
      </div>
      <van-field v-model="applyForm.shop_name" label="店铺名" placeholder="例如 阿强流动煎饼铺" />
      <van-field v-model="applyForm.contact_name" label="联系人" placeholder="联系人姓名" />
      <van-field v-model="applyForm.contact_phone" label="联系方式" placeholder="手机号/微信同号" />
      <van-field v-model="applyForm.category" label="摊位类型" placeholder="早餐小吃、咖啡饮品、夜宵烧烤等" />
      <van-field v-model="applyForm.photo_url" label="摊位照片" placeholder="图片 URL 或 data:image..." />
      <van-field v-model="applyForm.usual_area" label="常出摊区域" placeholder="地铁口、园区、社区、夜市" />
      <van-field v-model="applyForm.remark" label="补充说明" type="textarea" autosize placeholder="营业时段、主卖商品、证照情况" />
      <van-button block round color="#22C55E" :loading="submitting" @click="submitApplication">提交申请</van-button>
    </section>

    <section v-else class="card result-card">
      <div class="between">
        <div>
          <span class="pill green">下一步</span>
          <h2>{{ nextStepText }}</h2>
        </div>
        <van-button round plain @click="loadStatus">刷新状态</van-button>
      </div>
      <p class="muted">平台处理后，这里会同步展示通过、需补充、驳回或禁用原因。</p>
    </section>

    <section class="card merchant-flow-actions">
      <van-button round plain @click="logout">切换手机号</van-button>
      <van-button v-if="status?.next_action === 'application_approved'" round color="#60A5FA" @click="loadStatus">刷新进入工作台</van-button>
    </section>
  </main>

  <main v-else class="reference-host" aria-label="商家端参考实现">
    <iframe
      ref="frameEl"
      class="reference-frame"
      :src="frameSrc"
      title="摊主工作台"
      @load="applyMerchantRoute"
    ></iframe>
  </main>
</template>

<script setup>
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { showToast } from 'vant'
import { apiFetch, merchantHeaders, unifiedLogin } from '../../api/client'

const route = useRoute()
const frameEl = ref(null)
const token = ref(localStorage.getItem('merchant_token') || '')
const submitting = ref(false)
const status = ref(null)
const auth = reactive({ phone: localStorage.getItem('merchant_phone') || '13800138000', code: '123456' })
const applyForm = reactive({
  shop_name: '',
  contact_name: '',
  contact_phone: auth.phone,
  category: '',
  photo_url: '',
  usual_area: '',
  remark: ''
})
const frameSrc = computed(() => {
  const query = new URLSearchParams()
  if (import.meta.env.VITE_API_BASE) query.set('apiBase', import.meta.env.VITE_API_BASE)
  const raw = query.toString()
  return `/reference/merchant.html${raw ? `?${raw}` : ''}`
})
const application = computed(() => status.value?.application || null)
const showWorkbench = computed(() => token.value && status.value?.next_action === 'dashboard')
const needsApplicationForm = computed(() => ['apply', 'application_needs_info', 'application_rejected'].includes(status.value?.next_action || 'apply'))
const statusTitle = computed(() => ({
  apply: '先提交流动摊位申请',
  application_pending: '申请已提交，等待审核',
  application_needs_info: '需要补充资料',
  application_rejected: '申请未通过，可修改重提',
  application_approved: '审核已通过',
  disabled: '店铺已被禁用'
}[status.value?.next_action || 'apply'] || '申请状态') )
const statusDescription = computed(() => ({
  apply: '填写店铺名、联系人、联系方式、摊位类型和摊位照片后，平台会在后台审核。',
  application_pending: '平台正在处理你的申请，请保持电话畅通。',
  application_needs_info: '按处理说明补充资料后重新提交。',
  application_rejected: '查看驳回原因，修改资料后可以再次提交。',
  application_approved: '系统正在绑定店铺码，刷新后即可进入工作台。',
  disabled: status.value?.shop?.disabled_reason || '该店铺暂不可经营，请联系平台处理。'
}[status.value?.next_action || 'apply'] || '') )
const statusPillClass = computed(() => application.value?.status === 'rejected' ? 'red' : application.value?.status === 'approved' ? 'green' : '')
const nextStepText = computed(() => ({
  application_pending: '等待平台审核',
  application_approved: '刷新进入工作台',
  disabled: '联系平台申诉'
}[status.value?.next_action] || '查看处理结果'))

function screenFromPath(path) {
  if (path.includes('/orders')) return 'orders'
  if (path.includes('/products')) return 'products'
  return 'overview'
}

function fillApplicationForm(app = {}) {
  applyForm.shop_name = app.shop_name || applyForm.shop_name
  applyForm.contact_name = app.contact_name || applyForm.contact_name
  applyForm.contact_phone = app.contact_phone || auth.phone
  applyForm.category = app.category || applyForm.category
  applyForm.photo_url = app.photo_url || applyForm.photo_url
  applyForm.usual_area = app.usual_area || applyForm.usual_area
  applyForm.remark = app.remark || applyForm.remark
}

async function sendCode() {
  try {
    const resp = await apiFetch('/api/auth/sms/send', { method: 'POST', body: JSON.stringify({ phone: auth.phone, scene: 'merchant' }) })
    showToast(resp.dev_code ? `验证码 ${resp.dev_code}` : '验证码已发送')
  } catch (error) {
    showToast(error.message)
  }
}

async function login() {
  submitting.value = true
  try {
    const resp = await unifiedLogin('merchant', auth)
    localStorage.setItem('merchant_token', resp.token)
    localStorage.setItem('merchant_phone', auth.phone)
    token.value = resp.token
    status.value = resp
    fillApplicationForm(resp.application || { contact_phone: auth.phone })
    await nextTick()
    applyMerchantRoute()
  } catch (error) {
    showToast(error.message)
  } finally {
    submitting.value = false
  }
}

async function loadStatus() {
  if (!token.value) return
  try {
    const resp = await apiFetch('/api/merchant/applications/me', { headers: merchantHeaders() })
    status.value = resp
    fillApplicationForm(resp.application || { contact_phone: auth.phone })
    await nextTick()
    applyMerchantRoute()
  } catch (error) {
    showToast(error.message)
    if (/authorization|token|401/i.test(error.message)) logout()
  }
}

async function submitApplication() {
  if (!applyForm.photo_url) {
    showToast('请至少提供一张摊位照片')
    return
  }
  submitting.value = true
  try {
    const payload = JSON.stringify(applyForm)
    const resp = application.value?.id
      ? await apiFetch(`/api/merchant/applications/${application.value.id}`, { method: 'PUT', headers: merchantHeaders(), body: payload })
      : await apiFetch('/api/merchant-applications', { method: 'POST', body: payload })
    status.value = { ...(status.value || {}), application: resp.application, next_action: resp.next_action || 'application_pending' }
    showToast('申请已提交')
  } catch (error) {
    showToast(error.message)
  } finally {
    submitting.value = false
  }
}

function logout() {
  localStorage.removeItem('merchant_token')
  token.value = ''
  status.value = null
}

function applyMerchantRoute() {
  const frameWindow = frameEl.value?.contentWindow
  if (!frameWindow) return
  try {
    if (typeof frameWindow.setScreen === 'function') {
      frameWindow.setScreen(screenFromPath(route.path))
    }
  } catch (error) {
    // The reference file is same-origin in dev; ignore until iframe finishes booting.
  }
}

onMounted(() => {
  if (token.value) loadStatus()
})

watch(() => route.fullPath, async () => {
  await nextTick()
  applyMerchantRoute()
})
</script>
