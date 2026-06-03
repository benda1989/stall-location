import { apiFetch } from './client'

let configuredURL = ''

export function isWeChat() {
  return /MicroMessenger/i.test(window.navigator.userAgent)
}

export async function ensureWeChatJS() {
  if (!isWeChat() || !window.wx) {
    return false
  }
  if (configuredURL === window.location.href.split('#')[0]) return true
  const pageURL = encodeURIComponent(window.location.href.split('#')[0])
  const cfg = await apiFetch(`/api/wechat/js-config?url=${pageURL}`)
  if (!cfg.enabled) return false
  window.wx.config({
    debug: false,
    appId: cfg.app_id,
    timestamp: Number(cfg.timestamp),
    nonceStr: cfg.nonce_str,
    signature: cfg.signature,
    jsApiList: cfg.js_api_list
  })
  configuredURL = window.location.href.split('#')[0]
  return true
}

export async function openLocation({ lat, lng, name, address }) {
  if (await ensureWeChatJS()) {
    window.wx.ready(() => {
      window.wx.openLocation({ latitude: Number(lat), longitude: Number(lng), name, address, scale: 16 })
    })
    return
  }
  const url = `https://apis.map.qq.com/uri/v1/marker?marker=coord:${lat},${lng};title:${encodeURIComponent(name)};addr:${encodeURIComponent(address || '')}`
  window.location.href = url
}

export async function getLocation() {
  const devLocation = readDevLocation()
  if (devLocation && !isWeChat()) return devLocation
  if (await ensureWeChatJS()) {
    return new Promise((resolve, reject) => {
      window.wx.ready(() => {
        window.wx.getLocation({
          type: 'gcj02',
          success: (res) => resolve({ lat: res.latitude, lng: res.longitude, accuracy: Math.round(res.accuracy || 0) }),
          fail: reject
        })
      })
    })
  }
  return new Promise((resolve, reject) => {
    if (!navigator.geolocation) return reject(new Error('当前浏览器不支持定位'))
    navigator.geolocation.getCurrentPosition(
      (pos) => resolve({ lat: pos.coords.latitude, lng: pos.coords.longitude, accuracy: Math.round(pos.coords.accuracy) }),
      reject
    )
  })
}

function readDevLocation() {
  if (!import.meta.env.DEV) return null
  try {
    const saved = JSON.parse(localStorage.getItem('mplzDevLocation') || 'null')
    if (Number.isFinite(Number(saved?.lat)) && Number.isFinite(Number(saved?.lng))) {
      return { lat: Number(saved.lat), lng: Number(saved.lng), accuracy: Number(saved.accuracy || 20) }
    }
  } catch {}
  return null
}
