<template>
  <main class="page wide ops-redesign">
    <!-- <section class="ops-command-hero">
      <div class="ops-command-copy">
        <div class="eyebrow">Operations Command</div>
        <h1>经营后台</h1>
        <p>把入驻、商户、订单和出摊信号放在同一个经营视图里：先处理风险和待办，再看增长和履约。</p>
      </div>
      <div v-if="token" class="ops-duty-card">
        <span class="pill green">今日优先级</span>
        <strong>{{ topDutyCount }}</strong>
        <small>个待处理事项</small>
      </div>
    </section> -->

    <section v-if="!token" class="ops-login-card card form-stack">
      <div>
        <span class="pill green">Admin Access</span>
        <h2>手机号验证码登录</h2>
        <p class="muted">本地开发可使用后台手机号 13800139999，验证码 123456。</p>
      </div>
      <van-field v-model="form.phone" label="手机号" placeholder="后台手机号" />
      <div class="sms-row">
        <van-field v-model="form.code" label="验证码" placeholder="本地可用 123456" />
        <van-button round plain @click="sendCode">发送验证码</van-button>
      </div>
      <van-button block round color="#60A5FA" @click="loginBySMS">进入经营后台</van-button>
      <van-button block round plain @click="loginByDemo">开发账号 admin/admin123</van-button>
    </section>

    <template v-else>
      <section class="ops-stat-grid" aria-label="经营统计">
        <button v-for="card in statCards" :key="card.key" class="ops-stat-card" type="button" @click="activateStat(card)">
          <span>{{ card.label }}</span>
          <strong>{{ card.value }}</strong>
          <small>{{ card.hint }}</small>
          <i :class="card.tone"></i>
        </button>
      </section>

      <section class="ops-workbench">
        <aside class="ops-filter-rail" aria-label="后台导航">
          <button :class="{ active: activePanel === 'merchants' }" type="button" @click="activePanel = 'merchants'">商户管理</button>
          <button :class="{ active: activePanel === 'orders' }" type="button" @click="activePanel = 'orders'">订单管理</button>
          <button :class="{ active: activePanel === 'feedback' }" type="button" @click="activePanel = 'feedback'">反馈管理</button>
          <button :class="{ active: activePanel === 'sessions' }" type="button" @click="activePanel = 'sessions'">出摊地图</button>
          <button :class="{ active: activePanel === 'system' }" type="button" @click="activePanel = 'system'">系统管理</button>
        </aside>

        <div class="ops-main-stack">
          <section v-show="activePanel === 'merchants'" class="ops-section-card">
            <div class="between section-heading">
              <div class="segmented">
                <button :class="{ active: merchantFilter === 'needs_action' }" type="button" @click="merchantFilter = 'needs_action'">待处理</button>
                <button :class="{ active: merchantFilter === 'active' }" type="button" @click="merchantFilter = 'active'">正常</button>
                <button :class="{ active: merchantFilter === 'disabled' }" type="button" @click="merchantFilter = 'disabled'">禁用</button>
                <button :class="{ active: merchantFilter === 'all' }" type="button" @click="merchantFilter = 'all'">全部</button>
              </div>
            </div>

            <div class="ops-table merchant-table">
              <div class="ops-table-row head"><span>对象</span><span>状态</span><span>业务信息</span><span>排序原因</span><span>操作</span></div>
              <div v-for="row in filteredMerchantRows" :key="row.key" class="ops-table-row">
                <strong>{{ row.title }}<small>{{ row.code }}</small></strong>
                <span class="pill" :class="row.tone">{{ row.statusText }}</span>
                <span>{{ row.meta }}</span>
                <span class="muted">{{ row.reason }}</span>
                <span class="action-cluster">
                  <template v-if="row.kind === 'application'">
                    <van-button size="small" round color="#22C55E" @click="reviewApplication(row.raw, 'approve')">通过</van-button>
                    <van-button size="small" round plain @click="reviewApplication(row.raw, 'needs-info')">补充</van-button>
                    <van-button size="small" round plain @click="reviewApplication(row.raw, 'reject')">驳回</van-button>
                  </template>
                  <template v-else>
                    <van-button v-if="row.raw.status === 'disabled'" size="small" round color="#60A5FA" @click="enableShop(row.raw)">启用</van-button>
                    <van-button v-else size="small" round plain @click="disableShop(row.raw)">禁用</van-button>
                  </template>
                </span>
              </div>
              <div v-if="!filteredMerchantRows.length" class="empty">当前筛选下暂无商户</div>
            </div>
          </section>

          <section v-show="activePanel === 'orders'" class="ops-section-card">
            <div class="order-filter-bar">
              <div class="segmented">
                <button v-for="item in orderFilters" :key="item.value" :class="{ active: orderFilter === item.value }" type="button" @click="orderFilter = item.value">{{ item.label }}</button>
              </div>
              <div class="segmented compact">
                <button v-for="item in timeFilters" :key="item.value" :class="{ active: orderTimeFilter === item.value }" type="button" @click="orderTimeFilter = item.value">{{ item.label }}</button>
              </div>
            </div>

            <div class="ops-table order-table">
              <div class="ops-table-row head"><span>订单</span><span>商户/顾客</span><span>时间</span><span>状态</span><span>金额</span><span>操作</span></div>
              <div v-for="order in filteredOrders" :key="order.id" class="ops-table-row">
                <strong>{{ order.order_no }}<small>{{ order.items?.length || 0 }} 个商品</small></strong>
                <span>{{ order.shop?.name || '未知商户' }} · {{ order.customer_name }}</span>
                <span>{{ dateTime(order.created_at) }}</span>
                <span><b class="pill" :class="orderTone(order)">{{ orderStatusText(order) }}</b></span>
                <strong class="price">{{ money(order.total_amount_cents) }}</strong>
                <span class="action-cluster">
                  <van-button size="small" round plain :disabled="!canCancel(order)" @click="cancelOrder(order)">撤销</van-button>
                  <van-button size="small" round color="#FBBF24" :disabled="!canRefund(order)" @click="refundOrder(order)">退款</van-button>
                </span>
              </div>
              <div v-if="!filteredOrders.length" class="empty">当前筛选下暂无订单</div>
            </div>
          </section>

          <section v-show="activePanel === 'feedback'" class="ops-section-card">
            <div class="between section-heading">
              <div>
                <span class="pill green">Feedback Center</span>
                <h2>反馈管理</h2>
                <p class="muted">统一处理顾客端订单/定位反馈与商户端经营问题，待处理和处理中反馈优先展示。</p>
              </div>
              <van-button size="small" round color="#60A5FA" @click="load">刷新反馈</van-button>
            </div>

            <div class="order-filter-bar">
              <div class="segmented">
                <button v-for="item in feedbackFilters" :key="item.value" :class="{ active: feedbackFilter === item.value }" type="button" @click="feedbackFilter = item.value">{{ item.label }}</button>
              </div>
              <div class="segmented compact">
                <button v-for="item in feedbackSourceFilters" :key="item.value" :class="{ active: feedbackSourceFilter === item.value }" type="button" @click="feedbackSourceFilter = item.value">{{ item.label }}</button>
              </div>
            </div>

            <div class="ops-table feedback-table">
              <div class="ops-table-row head"><span>来源/联系人</span><span>反馈内容</span><span>关联对象</span><span>时间</span><span>状态</span><span>操作</span></div>
              <div v-for="item in filteredFeedback" :key="item.id" class="ops-table-row">
                <strong>{{ feedbackSourceText(item.source) }}<small>{{ item.contact_name || '匿名' }} · {{ item.contact_phone }}</small></strong>
                <span class="feedback-copy">{{ item.description }}<small>{{ item.page_url || '未记录页面' }}</small></span>
                <span>
                  {{ item.shop?.name || (item.shop_id ? `商户 #${item.shop_id}` : '未关联商户') }}
                  <small v-if="item.image_url"><a class="feedback-image-link" :href="item.image_url" target="_blank" rel="noreferrer">查看反馈图片</a></small>
                </span>
                <span>{{ dateTime(item.created_at) }}<small v-if="item.handled_at">处理于 {{ dateTime(item.handled_at) }}</small></span>
                <span><b class="pill" :class="feedbackTone(item.status)">{{ feedbackStatusText(item.status) }}</b><small v-if="item.handler_note">{{ item.handler_note }}</small></span>
                <span class="action-cluster">
                  <van-button v-if="item.status === 'pending'" size="small" round plain @click="updateFeedback(item, 'handling')">受理</van-button>
                  <van-button size="small" round color="#22C55E" :disabled="item.status === 'resolved'" @click="updateFeedback(item, 'resolved')">已处理</van-button>
                  <van-button size="small" round plain :disabled="item.status === 'closed'" @click="updateFeedback(item, 'closed')">关闭</van-button>
                  <van-button v-if="item.status !== 'pending'" size="small" round plain @click="updateFeedback(item, 'pending')">重开</van-button>
                </span>
              </div>
              <div v-if="!filteredFeedback.length" class="empty">当前筛选下暂无反馈</div>
            </div>
          </section>

          <section v-show="activePanel === 'sessions'" class="ops-section-card ops-map-console">
            <div class="between section-heading">
              <div>
                <span class="pill green">Tencent Live Map</span>
                <h2>正在营业商户位置</h2>
                <p class="muted">只展示已开始出摊且未到预计收摊时间的商户位置，用于后台运营巡检。</p>
              </div>
              <van-button size="small" round color="#60A5FA" @click="load">刷新点位</van-button>
            </div>

            <div class="ops-live-map-layout">
              <div class="ops-live-map-shell">
                <div ref="adminMapEl" class="admin-tencent-map" aria-label="腾讯地图展示正在营业商户位置"></div>
                <div v-if="adminMapStatus !== 'ready'" class="admin-map-fallback" aria-label="地图降级预览">
                  <div class="admin-map-grid"></div>
                  <button
                    v-for="row in activeSessionRows"
                    :key="`fallback-${row.id}`"
                    class="admin-map-fallback-pin"
                    :class="{ active: selectedSessionID === row.id }"
                    :style="fallbackPinStyle(row)"
                    type="button"
                    @click="focusSession(row)"
                  >
                    <strong>{{ shortShopName(row.name) }}</strong>
                    <small>{{ row.name }}</small>
                  </button>
                  <div class="admin-map-state-card">
                    <span class="pill" :class="adminMapStatus === 'error' ? 'red' : 'green'">{{ adminMapStatusText }}</span>
                    <strong>{{ activeSessionRows.length }} 个营业点位</strong>
                    <p>{{ adminMapNotice }}</p>
                  </div>
                </div>
                <div class="admin-map-overlay">
                  <span class="pill green">营业中 {{ activeSessionRows.length }}</span>
                  <strong>{{ selectedSession?.name || '全局点位' }}</strong>
                  <small>{{ selectedSession?.address || adminMapNotice }}</small>
                </div>
              </div>

              <aside class="ops-live-session-list" aria-label="正在营业商户列表">
                <button
                  v-for="row in activeSessionRows"
                  :key="row.id"
                  class="ops-live-session-row"
                  :class="{ active: selectedSessionID === row.id }"
                  type="button"
                  @click="focusSession(row)"
                >
                  <span class="pill green">营业中</span>
                  <strong>{{ row.name }}</strong>
                  <small>{{ row.category }} · {{ row.address }}</small>
                  <em>营业至 {{ dateTime(row.expected_end_at) }}</em>
                </button>
                <div v-if="!activeSessionRows.length" class="empty">暂无正在营业的商户点位</div>
              </aside>
            </div>
          </section>

          <section v-show="activePanel === 'system'" class="ops-section-card system-console">
            <div class="between section-heading">
              <div>
                <span class="pill green">System Users</span>
                <h2>系统用户</h2>
                <p class="muted">只维护后台登录用户的基础信息和启停状态。</p>
              </div>
              <van-button size="small" round color="#60A5FA" @click="createSystemUser">新增用户</van-button>
            </div>

            <div class="ops-table system-table">
              <div class="ops-table-row head"><span>用户</span><span>状态</span><span>更新时间</span><span>操作</span></div>
              <div v-for="user in systemUsers" :key="user.id" class="ops-table-row">
                <strong>{{ user.nickname }}<small>{{ user.phone }}</small></strong>
                <span class="pill" :class="user.status === 'disabled' ? 'red' : 'green'">{{ statusText(user.status) }}</span>
                <span>{{ dateTime(user.updated_at) }}</span>
                <span class="action-cluster">
                  <van-button size="small" round plain @click="editSystemUser(user)">编辑</van-button>
                  <van-button size="small" round :color="user.status === 'disabled' ? '#60A5FA' : '#F97316'" @click="toggleSystemUser(user)">{{ user.status === 'disabled' ? '启用' : '停用' }}</van-button>
                </span>
              </div>
              <div v-if="!systemUsers.length" class="empty">暂无系统用户</div>
            </div>
          </section>
        </div>
      </section>
    </template>
  </main>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { showToast } from 'vant'
import { adminHeaders, apiFetch, dateTime, money, unifiedLogin } from '../../api/client'
import { hasTencentMapKey, loadTencentMap, markerSvgDataUri } from '../../api/tencentMap'

const token = ref(localStorage.getItem('admin_token'))
const activePanel = ref(initialAdminPanel())
const merchantFilter = ref('needs_action')
const orderFilter = ref('all')
const orderTimeFilter = ref('all')
const orderSort = ref('desc')
const feedbackFilter = ref('open')
const feedbackSourceFilter = ref('all')
const form = reactive({ phone: '13800139999', code: '123456', username: 'admin', password: 'admin123' })
const shops = ref([])
const orders = ref([])
const sessions = ref([])
const applications = ref([])
const feedbacks = ref([])
const systemUsers = ref([])
const adminMapEl = ref(null)
const adminMapStatus = ref(hasTencentMapKey() ? 'idle' : 'missing-key')
const selectedSessionID = ref(null)

let adminMap = null
let adminMapSDK = null
let adminSessionMarkers = null

const orderFilters = [
  { value: 'all', label: '全部' },
  { value: 'pending_accept', label: '待接单' },
  { value: 'active', label: '处理中' },
  { value: 'ready', label: '待取货' },
  { value: 'canceled', label: '已撤销' },
  { value: 'refunded', label: '已退款' }
]
const timeFilters = [
  { value: 'today', label: '今天' },
  { value: '7d', label: '7 天' },
  { value: 'all', label: '全部时间' }
]
const feedbackFilters = [
  { value: 'open', label: '待处理' },
  { value: 'pending', label: '新反馈' },
  { value: 'handling', label: '处理中' },
  { value: 'resolved', label: '已处理' },
  { value: 'closed', label: '已关闭' },
  { value: 'all', label: '全部' }
]
const feedbackSourceFilters = [
  { value: 'all', label: '全部端' },
  { value: 'customer', label: '顾客端' },
  { value: 'merchant', label: '商户端' }
]

const pendingApplications = computed(() => applications.value.filter((app) => ['pending', 'reviewing', 'needs_info'].includes(app.status)).length)
const disabledShops = computed(() => shops.value.filter((shop) => shop.status === 'disabled').length)
const activeOrders = computed(() => orders.value.filter((order) => ['pending_accept', 'accepted', 'preparing', 'ready'].includes(order.status)).length)
const refundOrders = computed(() => orders.value.filter((order) => order.payment_status === 'refunded').length)
const pendingFeedback = computed(() => feedbacks.value.filter((item) => ['pending', 'handling'].includes(item.status)).length)
const todayAmountCents = computed(() => orders.value.filter((order) => isToday(order.created_at) && order.status !== 'canceled').reduce((sum, order) => sum + Number(order.total_amount_cents || 0), 0))
const topDutyCount = computed(() => pendingApplications.value + disabledShops.value + activeOrders.value + pendingFeedback.value)
const disabledSystemUsers = computed(() => systemUsers.value.filter((user) => user.status === 'disabled').length)
const shopsByID = computed(() => new Map(shops.value.map((shop) => [Number(shop.id), shop])))
const activeSessionRows = computed(() => sessions.value
  .map((session, index, rows) => normalizeSessionRow(session, index, rows))
  .filter((row) => row && Number.isFinite(Number(row.lat)) && Number.isFinite(Number(row.lng))))
const selectedSession = computed(() => activeSessionRows.value.find((row) => row.id === selectedSessionID.value) || activeSessionRows.value[0] || null)
const adminMapStatusText = computed(() => ({
  idle: '等待加载',
  loading: '加载腾讯地图',
  ready: '腾讯地图已连接',
  error: '地图加载失败',
  empty: '暂无点位',
  'missing-key': '未配置地图 Key'
})[adminMapStatus.value] || '地图待更新')
const adminMapNotice = computed(() => {
  if (!activeSessionRows.value.length) return '暂无正在营业的商户。'
  if (adminMapStatus.value === 'ready') return '可拖拽缩放地图，点击标记查看商户。'
  if (adminMapStatus.value === 'missing-key') return '未配置腾讯地图 Key，先展示降级点位预览。'
  if (adminMapStatus.value === 'error') return '腾讯地图 SDK 加载异常，保留点位预览和列表。'
  return '正在连接腾讯地图 SDK。'
})

const statCards = computed(() => [
  { key: 'pending-apps', label: '待处理入驻', value: pendingApplications.value, hint: '点击筛选商户管理', tone: 'warn', panel: 'merchants', filter: { merchantFilter: 'needs_action' } },
  { key: 'active-orders', label: '履约中订单', value: activeOrders.value, hint: '待接单/备货/待取货', tone: 'green', panel: 'orders', filter: { orderFilter: 'active' } },
  { key: 'pending-feedback', label: '待处理反馈', value: pendingFeedback.value, hint: '顾客/商户问题跟进', tone: 'warn', panel: 'feedback', filter: { feedbackFilter: 'open' } },
  { key: 'refund-orders', label: '退款订单', value: refundOrders.value, hint: '点击查看售后细节', tone: 'danger', panel: 'orders', filter: { orderFilter: 'refunded' } },
  { key: 'disabled-shops', label: '禁用商户', value: disabledShops.value, hint: '需要复核或启用', tone: 'danger', panel: 'merchants', filter: { merchantFilter: 'disabled' } },
  { key: 'live-sessions', label: '活跃出摊', value: sessions.value.length, hint: '实时公开点位', tone: 'green', panel: 'sessions', filter: {} },
])

const merchantRows = computed(() => {
  const appRows = applications.value.map((app) => ({
    key: `app-${app.id}`,
    kind: 'application',
    priority: applicationPriority(app),
    typeLabel: '入驻',
    title: app.shop_name,
    code: app.application_no || `#${app.id}`,
    statusText: applicationStatusText(app.status),
    tone: app.status === 'approved' ? 'green' : ['rejected', 'needs_info'].includes(app.status) ? 'red' : '',
    meta: `${app.contact_name} · ${app.contact_phone} · ${app.category || '未填品类'}`,
    reason: applicationReason(app),
    actionText: app.status === 'needs_info' ? '查看补充' : app.status === 'rejected' ? '查看驳回' : '立即审核',
    subtitle: `${app.contact_name} · ${app.usual_area || '未填常出摊区域'}`,
    raw: app
  }))
  const shopRows = shops.value.map((shop) => ({
    key: `shop-${shop.id}`,
    kind: 'shop',
    priority: shop.status === 'disabled' ? 15 : shop.verified_status !== 'verified' ? 25 : 80,
    typeLabel: '商户',
    title: shop.name,
    code: shop.shop_code,
    statusText: shopStatusText(shop),
    tone: shop.status === 'disabled' ? 'red' : shop.verified_status === 'verified' ? 'green' : '',
    meta: `${shop.category || '未填品类'} · ${shop.contact_phone || '未填电话'}`,
    reason: shop.status === 'disabled' ? (shop.disabled_reason || '已暂停经营') : shop.verified_status !== 'verified' ? '资料待完善' : '正常经营',
    actionText: shop.status === 'disabled' ? '启用' : '管理',
    subtitle: shop.status === 'disabled' ? (shop.disabled_reason || '禁用原因未记录') : shop.category,
    raw: shop
  }))
  return [...appRows, ...shopRows].sort((a, b) => a.priority - b.priority || String(b.raw.created_at || '').localeCompare(String(a.raw.created_at || '')))
})
const priorityRows = computed(() => merchantRows.value.filter((row) => row.priority < 50))
const filteredMerchantRows = computed(() => merchantRows.value.filter((row) => {
  if (merchantFilter.value === 'all') return true
  if (merchantFilter.value === 'needs_action') return row.priority < 50
  if (merchantFilter.value === 'disabled') return row.kind === 'shop' && row.raw.status === 'disabled'
  if (merchantFilter.value === 'active') return row.kind === 'shop' && row.raw.status !== 'disabled'
  return true
}))
const filteredOrders = computed(() => {
  const now = Date.now()
  const day = 24 * 60 * 60 * 1000
  return orders.value
    .filter((order) => {
      if (orderFilter.value === 'active') return ['pending_accept', 'accepted', 'preparing', 'ready'].includes(order.status)
      if (orderFilter.value === 'refunded') return order.payment_status === 'refunded'
      if (orderFilter.value !== 'all') return order.status === orderFilter.value
      return true
    })
    .filter((order) => {
      const t = new Date(order.created_at).getTime()
      if (orderTimeFilter.value === 'today') return isToday(order.created_at)
      if (orderTimeFilter.value === '7d') return now - t <= 7 * day
      return true
    })
    .sort((a, b) => orderSort.value === 'desc'
      ? new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
      : new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
})
const filteredFeedback = computed(() => feedbacks.value
  .filter((item) => {
    if (feedbackFilter.value === 'open') return ['pending', 'handling'].includes(item.status)
    if (feedbackFilter.value !== 'all') return item.status === feedbackFilter.value
    return true
  })
  .filter((item) => feedbackSourceFilter.value === 'all' || item.source === feedbackSourceFilter.value)
  .sort((a, b) => feedbackPriority(a) - feedbackPriority(b) || new Date(b.created_at).getTime() - new Date(a.created_at).getTime()))

async function sendCode() {
  try {
    const resp = await apiFetch('/api/auth/sms/send', { method: 'POST', body: JSON.stringify({ phone: form.phone, scene: 'admin' }) })
    showToast(resp.dev_code ? `验证码 ${resp.dev_code}` : '验证码已发送')
  } catch (error) {
    showToast(error.message)
  }
}
async function loginBySMS() { await login({ phone: form.phone, code: form.code }) }
async function loginByDemo() { await login({ username: form.username, password: form.password }) }
async function login(payload) {
  try {
    const resp = await unifiedLogin('admin', payload)
    localStorage.setItem('admin_token', resp.token)
    token.value = resp.token
    await load()
  } catch (error) {
    showToast(error.message)
  }
}
async function load() {
  const [applicationResp, shopResp, orderResp, feedbackResp, sessionResp, userResp] = await Promise.all([
    apiFetch('/api/admin/merchant-applications', { headers: adminHeaders() }),
    apiFetch('/api/admin/shops', { headers: adminHeaders() }),
    apiFetch('/api/admin/orders', { headers: adminHeaders() }),
    apiFetch('/api/admin/feedback', { headers: adminHeaders() }),
    apiFetch('/api/admin/stall-sessions/active', { headers: adminHeaders() }),
    apiFetch('/api/admin/system/users', { headers: adminHeaders() })
  ])
  applications.value = applicationResp.applications || []
  shops.value = shopResp.shops || []
  orders.value = orderResp.orders || []
  feedbacks.value = feedbackResp.feedback || []
  sessions.value = sessionResp.stall_sessions || []
  systemUsers.value = userResp.users || []
  if (!selectedSessionID.value && activeSessionRows.value[0]) selectedSessionID.value = activeSessionRows.value[0].id
}
function activateStat(card) { activatePanel(card.panel, card.filter) }
function activatePanel(panel, filters = {}) {
  activePanel.value = panel
  if (filters.merchantFilter) merchantFilter.value = filters.merchantFilter
  if (filters.orderFilter) orderFilter.value = filters.orderFilter
  if (filters.orderTimeFilter) orderTimeFilter.value = filters.orderTimeFilter
  if (filters.feedbackFilter) feedbackFilter.value = filters.feedbackFilter
  if (filters.feedbackSourceFilter) feedbackSourceFilter.value = filters.feedbackSourceFilter
}
function initialAdminPanel() {
  if (typeof window === 'undefined') return 'merchants'
  const panel = new URLSearchParams(window.location.search).get('panel')
  return ['merchants', 'orders', 'feedback', 'sessions', 'system'].includes(panel) ? panel : 'merchants'
}
function openPriority(row) {
  activePanel.value = row.kind === 'application' || row.kind === 'shop' ? 'merchants' : 'orders'
  if (row.kind === 'shop' && row.raw.status === 'disabled') merchantFilter.value = 'disabled'
  else merchantFilter.value = 'needs_action'
}
async function reviewApplication(app, action) {
  const defaultReason = action === 'approve' ? '资料完整，审核通过' : action === 'needs-info' ? '请补充清晰摊位照片和常出摊区域' : '暂不符合入驻要求'
  const reason = window.prompt('处理说明', app.review_reason || defaultReason)
  if (reason === null) return
  try {
    await apiFetch(`/api/admin/merchant-applications/${app.id}/${action}`, { method: 'POST', headers: adminHeaders(), body: JSON.stringify({ reason }) })
    showToast(action === 'approve' ? '已通过，店铺码已生成' : action === 'needs-info' ? '已要求补充资料' : '已驳回')
    await load()
  } catch (error) {
    showToast(error.message)
  }
}
async function disableShop(shop) {
  const reason = window.prompt('禁用原因', shop.disabled_reason || '异常经营，暂停接单')
  if (reason === null) return
  try {
    await apiFetch(`/api/admin/shops/${shop.id}/disable`, { method: 'POST', headers: adminHeaders(), body: JSON.stringify({ reason }) })
    showToast('已禁用')
    await load()
  } catch (error) {
    showToast(error.message)
  }
}
async function enableShop(shop) {
  try {
    await apiFetch(`/api/admin/shops/${shop.id}/enable`, { method: 'POST', headers: adminHeaders(), body: JSON.stringify({}) })
    showToast('已启用')
    await load()
  } catch (error) {
    showToast(error.message)
  }
}
async function cancelOrder(order) {
  const reason = window.prompt('撤销原因', '运营后台撤销订单')
  if (reason === null) return
  try {
    await apiFetch(`/api/admin/orders/${order.id}/cancel`, { method: 'POST', headers: adminHeaders(), body: JSON.stringify({ reason }) })
    showToast('订单已撤销')
    await load()
  } catch (error) {
    showToast(error.message)
  }
}
async function refundOrder(order) {
  const reason = window.prompt('退款原因', '运营后台退款')
  if (reason === null) return
  try {
    await apiFetch(`/api/admin/orders/${order.id}/refund`, { method: 'POST', headers: adminHeaders(), body: JSON.stringify({ reason }) })
    showToast('退款状态已更新')
    await load()
  } catch (error) {
    showToast(error.message)
  }
}

async function updateFeedback(item, status) {
  const noteDefaults = {
    pending: '重新打开，等待客服继续跟进',
    handling: item.handler_note || '已受理，正在跟进',
    resolved: item.handler_note || '已处理并回访',
    closed: item.handler_note || '无需继续处理，关闭反馈'
  }
  const note = window.prompt('处理备注', noteDefaults[status] || item.handler_note || '')
  if (note === null) return
  try {
    const resp = await apiFetch(`/api/admin/feedback/${item.id}`, { method: 'PUT', headers: adminHeaders(), body: JSON.stringify({ status, note }) })
    const index = feedbacks.value.findIndex((row) => row.id === item.id)
    if (index >= 0) feedbacks.value.splice(index, 1, resp.feedback)
    else await load()
    showToast('反馈状态已更新')
  } catch (error) {
    showToast(error.message)
  }
}

function statusText(status) { return status === 'disabled' ? '已停用' : '启用中' }
async function createSystemUser() {
  const phone = window.prompt('后台用户手机号')
  if (!phone) return
  const nickname = window.prompt('用户名称', '新后台用户')
  if (!nickname) return
  await saveSystemUser({ phone, nickname, status: 'active' }, 'POST', '/api/admin/system/users')
}
async function editSystemUser(user) {
  const phone = window.prompt('后台用户手机号', user.phone)
  if (!phone) return
  const nickname = window.prompt('用户名称', user.nickname)
  if (!nickname) return
  await saveSystemUser({ phone, nickname, status: user.status || 'active' }, 'PUT', `/api/admin/system/users/${user.id}`)
}
async function toggleSystemUser(user) {
  await saveSystemUser({ phone: user.phone, nickname: user.nickname, status: user.status === 'disabled' ? 'active' : 'disabled' }, 'PUT', `/api/admin/system/users/${user.id}`)
}
async function saveSystemUser(payload, method, url) {
  try {
    await apiFetch(url, { method, headers: adminHeaders(), body: JSON.stringify(payload) })
    showToast('系统用户已保存')
    await load()
  } catch (error) { showToast(error.message) }
}

function normalizeSessionRow(session, index, rows) {
  if (!session) return null
  const shop = session.shop || shopsByID.value.get(Number(session.shop_id)) || {}
  const projected = projectSessionToStage(session, index, rows)
  return {
    ...session,
    id: Number(session.id),
    name: shop.name || `商户 #${session.shop_id}`,
    category: shop.category || '未分组',
    contactPhone: shop.contact_phone || '',
    shopCode: shop.shop_code || '',
    lat: Number(session.lat),
    lng: Number(session.lng),
    x: projected.x,
    y: projected.y
  }
}
function projectSessionToStage(session, index, rows) {
  const valid = rows.filter((item) => Number.isFinite(Number(item.lat)) && Number.isFinite(Number(item.lng)))
  const avgLat = valid.reduce((sum, item) => sum + Number(item.lat), 0) / Math.max(valid.length, 1)
  const avgLng = valid.reduce((sum, item) => sum + Number(item.lng), 0) / Math.max(valid.length, 1)
  if (!Number.isFinite(Number(session.lat)) || !Number.isFinite(Number(session.lng))) {
    return { x: 24 + (index % 3) * 22, y: 30 + Math.floor(index / 3) * 18 }
  }
  return {
    x: Math.min(88, Math.max(12, 50 + (Number(session.lng) - avgLng) * 9000)),
    y: Math.min(82, Math.max(14, 50 - (Number(session.lat) - avgLat) * 9000))
  }
}
function fallbackPinStyle(row) {
  return { left: `${row.x}%`, top: `${row.y}%` }
}
function shortShopName(name = '') {
  const compact = String(name || '').replace(/\s+/g, '')
  return compact.slice(0, 2) || '摊'
}
function sessionMarkerColor(row) {
  const category = `${row.category || ''}${row.name || ''}`
  if (/咖啡|饮|茶|奶/.test(category)) return '#60A5FA'
  if (/水果|鲜|菜/.test(category)) return '#22C55E'
  if (/烤|炸|餐|粉|面|饭|煎/.test(category)) return '#F97316'
  if (/甜|糕|糖/.test(category)) return '#F43F5E'
  return '#FBBF24'
}
function sessionMapCenter(rows = activeSessionRows.value) {
  const valid = rows.filter((row) => Number.isFinite(Number(row.lat)) && Number.isFinite(Number(row.lng)))
  if (!valid.length) return { lat: 22.3193, lng: 114.1694 }
  return {
    lat: valid.reduce((sum, row) => sum + row.lat, 0) / valid.length,
    lng: valid.reduce((sum, row) => sum + row.lng, 0) / valid.length
  }
}
function clearAdminMapMarkers() {
  if (adminSessionMarkers) {
    adminSessionMarkers.setMap(null)
    adminSessionMarkers = null
  }
}
async function renderAdminMap(options = {}) {
  if (activePanel.value !== 'sessions') return
  await nextTick()
  const rows = activeSessionRows.value
  if (!rows.length) {
    clearAdminMapMarkers()
    adminMapStatus.value = 'empty'
    return
  }
  if (!selectedSessionID.value) selectedSessionID.value = rows[0].id
  if (!hasTencentMapKey()) {
    adminMapStatus.value = 'missing-key'
    return
  }
  if (!adminMapEl.value) return
  adminMapStatus.value = adminMap ? 'ready' : 'loading'
  try {
    adminMapSDK = await loadTencentMap()
    const centerRow = selectedSession.value || rows[0]
    const centerPoint = options.recenter && centerRow ? centerRow : sessionMapCenter(rows)
    const center = new adminMapSDK.LatLng(Number(centerPoint.lat), Number(centerPoint.lng))
    if (!adminMap) {
      adminMap = new adminMapSDK.Map(adminMapEl.value, { center, zoom: rows.length > 1 ? 16 : 17, viewMode: '2D' })
    } else if (options.recenter !== false) {
      adminMap.setCenter(center)
      if (typeof adminMap.setZoom === 'function') adminMap.setZoom(rows.length > 1 ? 16 : 17)
      if (typeof adminMap.resize === 'function') adminMap.resize()
    }
    clearAdminMapMarkers()
    adminSessionMarkers = new adminMapSDK.MultiMarker({
      map: adminMap,
      styles: Object.fromEntries(rows.map((row) => [`session-${row.id}`, new adminMapSDK.MarkerStyle({
        width: 76,
        height: 58,
        anchor: { x: 38, y: 56 },
        src: markerSvgDataUri({ fill: sessionMarkerColor(row), label: shortShopName(row.name) }),
        direction: 'bottom',
        offset: { x: 0, y: 8 },
        color: '#F8FAFC',
        size: 13,
        strokeColor: 'rgba(6, 17, 31, .78)',
        strokeWidth: 2,
        backgroundColor: 'rgba(6, 17, 31, .78)',
        backgroundBorderColor: 'rgba(255,255,255,.18)',
        backgroundBorderWidth: 1,
        backgroundBorderRadius: 999,
        padding: '5px 10px',
        wrapOptions: { maxWidth: 132, maxLineCount: 1 }
      })])),
      geometries: rows.map((row, index) => ({
        id: String(row.id),
        styleId: `session-${row.id}`,
        position: new adminMapSDK.LatLng(row.lat, row.lng),
        content: row.name,
        rank: 1000 - index,
        properties: { title: row.name, address: row.address }
      }))
    })
    adminSessionMarkers.on('click', (event) => {
      const id = Number(event.geometry?.id)
      const row = activeSessionRows.value.find((item) => item.id === id)
      if (row) focusSession(row)
    })
    adminMapStatus.value = 'ready'
  } catch (error) {
    clearAdminMapMarkers()
    adminMapStatus.value = 'error'
  }
}
function focusSession(row) {
  selectedSessionID.value = row.id
  if (adminMap && adminMapSDK && Number.isFinite(row.lat) && Number.isFinite(row.lng)) {
    adminMap.setCenter(new adminMapSDK.LatLng(row.lat, row.lng))
    if (typeof adminMap.setZoom === 'function') adminMap.setZoom(17)
  }
}

function applicationPriority(app) {
  if (app.status === 'pending' || app.status === 'reviewing') return 10
  if (app.status === 'needs_info') return 20
  if (app.status === 'rejected') return 45
  return 90
}
function applicationReason(app) {
  if (app.status === 'pending') return '新申请待审核'
  if (app.status === 'reviewing') return '运营处理中'
  if (app.status === 'needs_info') return app.review_reason || '等待申请人补充资料'
  if (app.status === 'rejected') return app.review_reason || '已驳回，可复盘原因'
  return '已完成入驻'
}
function applicationStatusText(status) {
  return { pending: '待审核', reviewing: '审核中', needs_info: '需补充', approved: '已通过', rejected: '未通过' }[status] || status || '未知'
}
function shopStatusText(shop) {
  if (shop.status === 'disabled') return '已禁用'
  if (shop.verified_status === 'verified') return '正常经营'
  return '资料待完善'
}
function orderStatusText(order) {
  if (order.payment_status === 'refunded') return '已退款'
  return {
    pending_accept: '待接单', accepted: '已接单', preparing: '备货中', ready: '待取货', completed: '已完成', rejected: '已拒单', canceled: '已撤销', expired: '已过期'
  }[order.status] || order.status || '未知'
}
function orderTone(order) {
  if (order.payment_status === 'refunded') return 'red'
  if (['ready', 'completed'].includes(order.status)) return 'green'
  if (['canceled', 'rejected', 'expired'].includes(order.status)) return 'red'
  return ''
}
function feedbackPriority(item) {
  return { pending: 0, handling: 1, resolved: 2, closed: 3 }[item.status] ?? 9
}
function feedbackSourceText(source) {
  return source === 'merchant' ? '商户端反馈' : '顾客端反馈'
}
function feedbackStatusText(status) {
  return { pending: '待处理', handling: '处理中', resolved: '已处理', closed: '已关闭' }[status] || status || '未知'
}
function feedbackTone(status) {
  if (status === 'resolved') return 'green'
  if (status === 'closed') return 'red'
  return ''
}
function canCancel(order) { return ['pending_accept', 'accepted', 'preparing'].includes(order.status) }
function canRefund(order) { return order.payment_status === 'paid' && order.status !== 'canceled' }
function isToday(value) {
  const d = new Date(value)
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

onMounted(() => {
  if (token.value) load().catch((error) => {
    showToast(error.message)
    localStorage.removeItem('admin_token')
    token.value = ''
  })
})

watch(activePanel, (panel) => {
  if (panel === 'sessions') renderAdminMap({ recenter: true })
})

watch(activeSessionRows, () => {
  if (activePanel.value === 'sessions') renderAdminMap({ recenter: false })
})

onBeforeUnmount(() => {
  clearAdminMapMarkers()
  if (adminMap && typeof adminMap.destroy === 'function') adminMap.destroy()
})
</script>
