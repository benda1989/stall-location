let loadPromise

export function hasTencentMapKey() {
  return Boolean(import.meta.env.VITE_TENCENT_MAP_KEY)
}

export function loadTencentMap() {
  if (window.TMap) return Promise.resolve(window.TMap)
  const key = import.meta.env.VITE_TENCENT_MAP_KEY
  if (!key) return Promise.reject(new Error('missing VITE_TENCENT_MAP_KEY'))
  if (loadPromise) return loadPromise
  loadPromise = new Promise((resolve, reject) => {
    const script = document.createElement('script')
    script.charset = 'utf-8'
    script.async = true
    script.src = `https://map.qq.com/api/gljs?v=1.exp&key=${encodeURIComponent(key)}`
    script.onload = () => window.TMap ? resolve(window.TMap) : reject(new Error('Tencent map loaded without TMap'))
    script.onerror = () => reject(new Error('Tencent map script failed'))
    document.head.appendChild(script)
  })
  return loadPromise
}

export function markerSvgDataUri({ fill = '#60A5FA', stroke = '#06111F', label = '' }) {
  const safeLabel = String(label || '').slice(0, 3)
  const svg = `
  <svg xmlns="http://www.w3.org/2000/svg" width="76" height="58" viewBox="0 0 76 58">
    <filter id="shadow" x="-30%" y="-30%" width="160%" height="180%"><feDropShadow dx="0" dy="8" stdDeviation="6" flood-color="#000" flood-opacity="0.34"/></filter>
    <g filter="url(#shadow)">
      <path d="M8 8C8 3.58 11.58 0 16 0h44c4.42 0 8 3.58 8 8v24c0 4.42-3.58 8-8 8H43.5L36 55l-7.5-15H16c-4.42 0-8-3.58-8-8V8Z" fill="${fill}" stroke="${stroke}" stroke-width="3"/>
      <text x="38" y="25" text-anchor="middle" font-family="Arial, sans-serif" font-weight="800" font-size="15" fill="${stroke}">${safeLabel}</text>
    </g>
  </svg>`
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}
