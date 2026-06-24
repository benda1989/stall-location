export function getUserLocation() {
  return new Promise((resolve, reject) => {
    uni.getLocation({
      type: 'gcj02',
      isHighAccuracy: true,
      success: (res) => resolve({ lat: Number(res.latitude), lng: Number(res.longitude), accuracy: Math.round(Number(res.accuracy || 0)) }),
      fail: (err) => reject(new Error(err?.errMsg || '定位失败'))
    })
  })
}

export function openVendorLocation(vendor) {
  const latitude = Number(vendor?.lat)
  const longitude = Number(vendor?.lng)
  if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
    uni.showToast({ title: '暂无摊位位置', icon: 'none' })
    return
  }
  uni.openLocation({
    latitude,
    longitude,
    name: vendor?.name || '流动摊位',
    address: vendor?.address || vendor?.area || '摊主当前位置',
    scale: 16
  })
}

export function distanceMeters(from, to) {
  const lat1 = Number(from?.lat)
  const lng1 = Number(from?.lng)
  const lat2 = Number(to?.lat)
  const lng2 = Number(to?.lng)
  if (![lat1, lng1, lat2, lng2].every(Number.isFinite)) return NaN
  const rad = Math.PI / 180
  const dLat = (lat2 - lat1) * rad
  const dLng = (lng2 - lng1) * rad
  const a = Math.sin(dLat / 2) ** 2 + Math.cos(lat1 * rad) * Math.cos(lat2 * rad) * Math.sin(dLng / 2) ** 2
  return 6371000 * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

export function distanceText(vendor, userLocation) {
  const preferred = Number(vendor?.distanceMeters)
  const meters = Number.isFinite(preferred) ? preferred : distanceMeters(userLocation, vendor)
  if (!Number.isFinite(meters)) return vendor?.endText ? `营业至 ${vendor.endText}` : '附近出摊'
  const distance = meters >= 1000 ? `${(meters / 1000).toFixed(meters >= 10000 ? 0 : 1)}km` : `${Math.max(1, Math.round(meters))}m`
  const walk = Number(vendor?.walkMinutes)
  return Number.isFinite(walk) && walk > 0 ? `${distance} · 步行${walk}分钟` : distance
}

export function hasFiniteLocation(location) {
  return Number.isFinite(Number(location?.lat)) && Number.isFinite(Number(location?.lng))
}
