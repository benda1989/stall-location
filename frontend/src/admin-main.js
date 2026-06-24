import { createApp } from 'vue'
import Vant from 'vant'
import 'vant/lib/index.css'
import AdminConsole from './views/admin/AdminConsole.vue'

createApp(AdminConsole).use(Vant).mount('#admin-app')
