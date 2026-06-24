export const STALL_CATEGORY_OPTIONS = ['早餐小吃', '咖啡饮品', '水果鲜切', '夜宵烧烤', '便当快餐', '甜品冷饮', '卤味熟食', '鲜花手作', '生鲜菜摊', '其他摊位']

const CATEGORY_CATALOG = {
  breakfast: {
    family: 'meal',
    aliases: ['早餐', '小吃', '早餐小吃'],
    marker: { pin: '#F97316', ink: '#3A1F0A', paper: '#FFF7ED' },
    body: '<path class="sticker-fill" d="M9 15.2c.9-4 4-6.4 7-6.4s6.1 2.4 7 6.4H9Z"/><path class="sticker-line" d="M8 15.2h16l-1.1 8.1a2.3 2.3 0 0 1-2.3 2H11.4a2.3 2.3 0 0 1-2.3-2L8 15.2Z"/><path class="sticker-line" d="M13 19.5h6"/>'
  },
  coffee: {
    family: 'drink',
    aliases: ['咖啡', '饮品', '咖啡饮品'],
    marker: { pin: '#2563EB', ink: '#102A56', paper: '#EFF6FF' },
    body: '<path class="sticker-fill" d="M10 13h12v7.1a5 5 0 0 1-5 5h-2a5 5 0 0 1-5-5V13Z"/><path class="sticker-line" d="M22 15h1.4a2.8 2.8 0 1 1 0 5.6H22"/><path class="sticker-line" d="M13 9V6.8M18 9V6.8"/>'
  },
  fruit: {
    family: 'fresh',
    aliases: ['水果', '鲜切', '水果鲜切'],
    marker: { pin: '#16A34A', ink: '#0F3E22', paper: '#F0FDF4' },
    body: '<path class="sticker-fill" d="M15.2 12.2c4.7 0 8 3.2 8 7.2 0 4.2-3.5 6.8-7.8 6.8s-7.8-2.6-7.8-6.8c0-4 3.1-7.2 7.6-7.2Z"/><path class="sticker-line" d="M15.1 12.1c.9-3.2 2.8-5.1 5.7-5.8"/><path class="sticker-line" d="M18.8 7.6c1.7-.7 3.4-.6 4.6.4-1.3 1.7-3.1 2.1-5.2 1.4"/>'
  },
  barbecue: {
    family: 'meal',
    aliases: ['夜宵', '烧烤', '夜宵烧烤', '夜宵小吃'],
    marker: { pin: '#EA580C', ink: '#3B1607', paper: '#FFF7ED' },
    body: '<path class="sticker-fill" d="M13.6 8.3c3.9 3 5.7 6.1 5.2 9.2-.4 2.3-2.1 4-4.7 4.9.8-2.5.4-4.4-1.2-5.7-1.8 1.9-2.5 4.1-2 6.7-2.6-1.7-3.8-3.8-3.4-6.3.4-2.8 2.4-5.8 6.1-8.8Z"/><path class="sticker-line" d="M21.8 9.3 10.4 25.5"/><path class="sticker-line" d="M18 10.3l4.8 3.3M15.5 14.2l4.4 3.1M13 18.1l4 2.8"/>'
  },
  bento: {
    family: 'meal',
    aliases: ['便当', '快餐', '便当快餐'],
    marker: { pin: '#D97706', ink: '#3A2205', paper: '#FFFBEB' },
    body: '<rect class="sticker-fill" x="7.5" y="11" width="17" height="13.5" rx="3"/><path class="sticker-line" d="M10.5 11V8.6h11V11M7.8 16.5h16.4M14 16.5v7.8M10.6 20.2h3.3M18.4 20.2h3.1"/>'
  },
  dessert: {
    family: 'treat',
    aliases: ['甜品', '冷饮', '甜品冷饮'],
    marker: { pin: '#E11D48', ink: '#4A1020', paper: '#FFF1F2' },
    body: '<path class="sticker-fill" d="M9.2 15.8h13.6l-1.6 8.1a2.6 2.6 0 0 1-2.5 2H13.3a2.6 2.6 0 0 1-2.5-2l-1.6-8.1Z"/><path class="sticker-line" d="M11.8 15.7c.2-3.4 2-5.5 4.2-5.5s4 2.1 4.2 5.5"/><path class="sticker-line" d="M14.5 9.9c-.4-1.9.5-3.4 2-4.3"/>'
  },
  braise: {
    family: 'braise',
    aliases: ['卤味', '熟食', '卤味熟食'],
    marker: { pin: '#B45309', ink: '#3F2107', paper: '#FFFBEB' },
    body: '<path class="sticker-fill" d="M8.5 14h15l-1.2 9.4a2.5 2.5 0 0 1-2.5 2.2h-7.6a2.5 2.5 0 0 1-2.5-2.2L8.5 14Z"/><path class="sticker-line" d="M11.5 14v-2.4a4.5 4.5 0 0 1 9 0V14"/><path class="sticker-line" d="M12.6 18.8h6.8M16 6.2v2"/>'
  },
  flower: {
    family: 'craft',
    aliases: ['鲜花', '手作', '鲜花手作'],
    marker: { pin: '#DB2777', ink: '#4A1530', paper: '#FDF2F8' },
    body: '<path class="sticker-fill" d="M16 14c-2.6-3.2-1.8-7.1.7-8.7 2.6 1.6 3.4 5.5.8 8.7"/><path class="sticker-line" d="M15.7 14c-4-.5-6.4-3.3-6-6.3 2.9-.9 6.4.8 7.4 4"/><path class="sticker-line" d="M16.3 14c3.9-.4 6.2-2.8 6.1-5.8-2.6-1.1-6 .2-7.2 3.3"/><path class="sticker-line" d="M16 14.5v11M12.2 19.4 16 22.1l3.8-2.7"/>'
  },
  vegetable: {
    family: 'fresh',
    aliases: ['生鲜', '菜摊', '生鲜菜摊'],
    marker: { pin: '#059669', ink: '#073B2A', paper: '#ECFDF5' },
    body: '<path class="sticker-fill" d="M8.5 15.6h15l-1.3 8.6a2.4 2.4 0 0 1-2.4 2H12.2a2.4 2.4 0 0 1-2.4-2l-1.3-8.6Z"/><path class="sticker-line" d="M7.5 15.6h17"/><path class="sticker-line" d="M11.7 15.2c.6-3.8 3.5-6.4 7.7-7.8.8 4.1-.7 7-4.6 8.7"/><path class="sticker-line" d="M13.5 14.1c1.4-.7 2.6-1.6 3.6-2.9"/>'
  },
  other: {
    family: 'neutral',
    aliases: ['其他', '其他摊位', '杂货', '流动摊位'],
    marker: { pin: '#64748B', ink: '#1F2937', paper: '#F8FAFC' },
    body: '<path class="sticker-fill" d="M9 13.2h14v11.2H9V13.2Z"/><path class="sticker-line" d="M11.2 13.2V10a4.8 4.8 0 0 1 9.6 0v3.2M12.5 17.6h7M12.5 21h4.8"/><path class="sticker-line" d="M7.5 24.4h17"/>'
  }
}

const CATEGORY_ALIASES = Object.create(null)
Object.entries(CATEGORY_CATALOG).forEach(([key, config]) => {
  CATEGORY_ALIASES[key] = key
  config.aliases.forEach((alias) => { CATEGORY_ALIASES[alias] = key })
})

function encodeSvg(svg) {
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}

function categoryKey(category = '') {
  const label = String(category || '').trim()
  if (!label) return 'other'
  if (CATEGORY_CATALOG[label]) return label
  if (CATEGORY_ALIASES[label]) return CATEGORY_ALIASES[label]
  const compact = label.replace(/\s+/g, '')
  if (CATEGORY_ALIASES[compact]) return CATEGORY_ALIASES[compact]
  const matched = Object.entries(CATEGORY_CATALOG).find(([, config]) =>
    config.aliases.some((alias) => compact.includes(alias) || alias.includes(compact))
  )
  return matched?.[0] || 'other'
}

export function categoryConfig(category = '') {
  return CATEGORY_CATALOG[categoryKey(category)] || CATEGORY_CATALOG.other
}

export function categoryFamily(category = '') {
  return categoryConfig(category).family
}

export function categoryIconSvg(category = '') {
  const config = categoryConfig(category)
  const marker = config.marker || CATEGORY_CATALOG.other.marker
  return `<svg class="category-sticker" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" aria-hidden="true"><style>.sticker-paper{fill:${marker.paper};stroke:rgba(47,31,13,.24);stroke-width:1.2}.sticker-fill{fill:${marker.pin};fill-opacity:.2;stroke:${marker.ink};stroke-width:2.1;stroke-linecap:round;stroke-linejoin:round}.sticker-line{fill:none;stroke:${marker.ink};stroke-width:2.2;stroke-linecap:round;stroke-linejoin:round}</style><rect class="sticker-paper" x="3.8" y="3.8" width="24.4" height="24.4" rx="8.2" transform="rotate(-4 16 16)"/><g>${config.body}</g></svg>`
}

export function categoryIconDataUri(category = '') {
  return encodeSvg(categoryIconSvg(category))
}

export function tencentMerchantMarkerDataUri(category = '') {
  const config = categoryConfig(category)
  const marker = config.marker || CATEGORY_CATALOG.other.marker
  return encodeSvg(`<svg width="44" height="52" viewBox="0 0 44 52" fill="none" xmlns="http://www.w3.org/2000/svg"><style>.sticker-fill{fill:${marker.pin};fill-opacity:.2;stroke:${marker.ink};stroke-width:2.1;stroke-linecap:round;stroke-linejoin:round}.sticker-line{fill:none;stroke:${marker.ink};stroke-width:2.2;stroke-linecap:round;stroke-linejoin:round}</style><path d="M22 50C22 50 40 33.4 40 19.8C40 9.4 32 2 22 2C12 2 4 9.4 4 19.8C4 33.4 22 50 22 50Z" fill="${marker.pin}" stroke="#FFF8E7" stroke-width="3"/><circle cx="22" cy="20" r="12.5" fill="${marker.paper}"/><svg x="9" y="7" width="26" height="26" viewBox="0 0 32 32">${config.body}</svg></svg>`)
}


export function categoryColor(category = '') {
  const config = categoryConfig(category)
  return config?.marker?.pin || '#64748B'
}

export function categoryInitial(category = '') {
  return String(category || '摊').slice(0, 1)
}
