import { createRouter, createWebHistory } from 'vue-router'
import Home from '../views/Home.vue'
import CustomerReference from '../views/customer/CustomerReference.vue'
import MerchantReference from '../views/merchant/MerchantReference.vue'
import AdminConsole from '../views/admin/AdminConsole.vue'

const merchantRoutes = [
  '/merchant/login',
  '/merchant/apply',
  '/merchant/application-status',
  '/merchant/onboarding',
  '/merchant/dashboard',
  '/merchant/stall-session',
  '/merchant/products',
  '/merchant/orders',
  '/merchant/qrcode'
]

const routes = [
  { path: '/', redirect: '/nearby' },
  { path: '/home', component: Home },
  {
    path: '/reference/customer.html',
    redirect: (to) => {
      if (to.query.shareCode) return { path: `/share/${encodeURIComponent(to.query.shareCode)}`, query: { preview: to.query.preview } }
      if (to.query.shopCode) return { path: `/s/${encodeURIComponent(to.query.shopCode)}`, query: { preview: to.query.preview } }
      return { path: '/nearby', query: { preview: to.query.preview } }
    }
  },
  { path: '/reference/merchant.html', redirect: '/merchant/dashboard' },
  { path: '/nearby', component: CustomerReference },
  { path: '/share/:shareCode', component: CustomerReference },
  { path: '/s/:shopCode', component: CustomerReference },
  { path: '/merchant', redirect: '/merchant/dashboard' },
  ...merchantRoutes.map((path) => ({ path, component: MerchantReference })),
  { path: '/admin', component: AdminConsole }
]

export default createRouter({ history: createWebHistory(), routes })
