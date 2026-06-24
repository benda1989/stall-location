<template>
  <view :class="['customer-page', `mode-${appMode}`, activeView === 'map' && appMode === 'customer' && 'is-map-tab']" :style="pageStyle">
    <view class="customer-page-title">{{ appTitle }}</view>

    <view v-if="appMode === 'application'" class="merchant-flow-page">
      <view class="merchant-panel merchant-status-card">
        <text class="merchant-status-label">当前状态</text>
        <view class="merchant-status-row">
          <text :class="['merchant-status-pill', applicationStatusClass]">{{ applicationStatusText }}</text>
        </view>
        <view v-if="merchantApplication?.review_reason" class="merchant-review-reason">
          <text class="merchant-review-label">说明</text>
          <text class="merchant-muted">{{ merchantApplication.review_reason }}</text>
        </view>
      </view>

      <view v-if="merchantApplication && !needsApplicationForm" class="merchant-panel application-info-card">
        <view class="merchant-section-head">
          <text class="merchant-section-title">已填写资料</text>
        </view>
        <image v-if="merchantApplication.photo_url" class="application-stall-photo" :src="merchantApplication.photo_url" mode="aspectFill" />
        <view class="application-info-list">
          <view v-for="item in applicationInfoRows" :key="item.label" class="application-info-row">
            <text class="application-info-label">{{ item.label }}</text>
            <text class="application-info-value">{{ item.value }}</text>
          </view>
        </view>
      </view>

      <view v-if="needsApplicationForm" class="merchant-panel merchant-form-card">
        <view class="merchant-section-head">
          <text class="merchant-section-title">{{ merchantApplication?.id ? '修改并重新提交' : '填写入驻资料' }}</text>
          <text class="merchant-muted">{{ merchantApplication?.id ? '修改资料后平台会重新审核' : '提交后再次打开会进入申请状态页' }}</text>
        </view>
        <view class="simple-form merchant-simple-form">
          <input v-model="merchantApplyForm.merchant_name" placeholder="摊位/商户名" />
          <input v-model="merchantApplyForm.contact_name" placeholder="联系人" />
          <input v-model="merchantApplyForm.contact_phone" type="number" placeholder="联系方式" />
          <picker mode="selector" :range="categories" :value="merchantApplyCategoryIndex" @change="chooseMerchantApplyCategory">
            <view :class="['form-picker', !merchantApplyForm.category && 'is-placeholder']">{{ merchantApplyForm.category || '请选择摊位类型' }}</view>
          </picker>
          <view class="merchant-upload-row">
            <input v-model="merchantApplyForm.photo_url" placeholder="摊位照片 URL（可选）" />
            <button class="merchant-mini-btn" :loading="merchantLoading" @tap="chooseMerchantApplyPhoto">上传</button>
          </view>
          <image v-if="merchantApplyForm.photo_url" class="merchant-image-preview" :src="merchantApplyForm.photo_url" mode="aspectFill" />
          <input v-model="merchantApplyForm.usual_area" placeholder="常出摊区域" />
          <textarea v-model="merchantApplyForm.remark" placeholder="补充说明：营业时段、主卖商品等" />
          <button class="submit-order" :loading="submittingApplication" @tap="submitApplication">提交申请</button>
        </view>
      </view>

      <view v-else class="merchant-panel merchant-result-card next-step-card">
        <text class="merchant-section-title">下一步</text>
        <text class="merchant-muted">{{ merchantNextStepText }}</text>
      </view>

      <button class="identity-switch" @tap="switchAppMode('customer')">顾客端</button>
    </view>

    <view v-else-if="appMode === 'merchant'" class="merchant-workbench-page">
      <view class="merchant-tabs">
        <button :class="merchantTab === 'overview' && 'is-active'" @tap="switchMerchantTab('overview')">总览</button>
        <button :class="merchantTab === 'products' && 'is-active'" @tap="switchMerchantTab('products')">商品</button>
      </view>

      <view v-if="merchantTab === 'overview'" class="merchant-grid">
        <view :class="['merchant-hero-card', activeMerchantSession ? 'is-live' : 'is-idle']">
          <image v-if="activeMerchantSession?.photo_url" class="merchant-hero-bg" :src="activeMerchantSession.photo_url" mode="aspectFill" />
          <view v-if="activeMerchantSession" class="merchant-hero-live-copy">
            <text class="merchant-live-pill">出摊中</text>
            <text class="merchant-hero-title">截止 {{ timeText(activeMerchantSession.expected_end_at) }}</text>
            <text v-if="merchantSessionAddress" class="merchant-hero-copy">{{ merchantSessionAddress }}</text>
          </view>
          <button class="merchant-hero-action" :loading="merchantSessionLoading" @tap="activeMerchantSession ? endMerchantSession() : openSessionSheet()">
            {{ activeMerchantSession ? '结束出摊' : '出摊' }}
          </button>
        </view>

        <view class="merchant-panel merchant-card">
          <view class="merchant-card-head">
            <view class="merchant-card-title-block">
              <text class="merchant-section-title">{{ merchantDisplayName }}</text>
              <text class="merchant-muted">{{ merchantDashboard?.merchant?.category || merchantStatus?.merchant?.category || '摊位类型待完善' }}</text>
            </view>
          </view>
          <image v-if="merchantQrDataUrl" class="merchant-qr" :src="merchantQrDataUrl" mode="aspectFit" @tap="previewMerchantQr" />
          <view v-else class="merchant-qr-placeholder"><text>分享二维码生成中</text></view>
        </view>
      </view>

      <view v-else class="merchant-grid">
        <view v-if="merchantProducts.length" class="merchant-product-list">
          <view v-for="product in merchantProducts" :key="product.id" class="merchant-product-row">
            <button
              v-if="shouldShowPinControl(product)"
              :class="['merchant-pin-btn', 'is-floating', isProductPinned(product) ? 'is-pinned' : 'is-empty']"
              :aria-label="isProductPinned(product) ? '取消置顶' : '置顶'"
              @tap.stop="toggleProductPinned(product)"
            >
              <image class="merchant-pin-icon" :src="pinIconDataUri(isProductPinned(product))" mode="aspectFit" />
            </button>
            <view class="merchant-product-thumb" @tap="chooseProductImage(product)">
              <image v-if="product.image_url" :src="product.image_url" mode="aspectFill" />
              <text v-else>{{ productInitial(product) }}</text>
              <text class="merchant-product-thumb-edit">更换</text>
            </view>
            <view class="merchant-product-main">
              <view class="merchant-product-head">
                <input v-if="isProductEditing(product, 'name')" v-model="product.name" class="merchant-inline-input name" focus @blur="commitProductEdit(product)" />
                <button v-else class="merchant-inline-value merchant-product-name-button" @tap="startProductEdit(product, 'name')">{{ product.name }}</button>
              </view>
              <view class="merchant-product-price-row">
                <input v-if="isProductEditing(product, 'price')" :value="productPriceYuan(product)" class="merchant-inline-input price" type="digit" focus @blur="updateProductPrice(product, $event)" />
                <button v-else class="merchant-inline-value merchant-price-button" @tap="startProductEdit(product, 'price')">¥{{ productPriceYuan(product) }}</button>
              </view>
              <view class="merchant-product-actions">
                <button :class="['merchant-mini-btn', product.status === 'on_sale' && 'is-on']" @tap="toggleProduct(product)">{{ product.status === 'on_sale' ? '上架中' : '已下架' }}</button>
                <button v-if="product.status === 'off_sale'" class="merchant-mini-btn danger" @tap="deleteProduct(product)">删除</button>
              </view>
            </view>
          </view>
        </view>
        <view v-else-if="merchantProductsLoading" class="state-card merchant-product-list-state"><text>正在加载商品…</text></view>
        <view v-else class="state-card merchant-product-list-state empty-card"><text>暂无商品，点击右下角 + 添加。</text></view>
        <view v-if="merchantProducts.length" class="merchant-product-list-state">
          <text>{{ merchantProductsLoading ? '正在加载更多…' : (merchantProductsFinished ? '没有更多商品了' : '继续下滑加载更多') }}</text>
        </view>
      </view>

      <button v-if="merchantTab === 'products'" class="merchant-add-fab" @tap="productFormOpen = !productFormOpen">+</button>
      <button class="identity-switch" @tap="switchAppMode('customer')">顾客端</button>

      <view v-if="productFormOpen" class="sheet-mask" @tap="productFormOpen = false">
        <view class="bottom-sheet merchant-product-form-sheet" @tap.stop>
          <view class="sheet-handle"></view>
          <view class="sheet-head">
            <view>
              <text class="sheet-title">添加商品</text>
              <text class="sheet-subtitle">商品会展示在顾客看到的摊铺详情里</text>
            </view>
            <button class="close-button" @tap="productFormOpen = false">关闭</button>
          </view>
          <view class="simple-form merchant-simple-form">
            <input v-model="productForm.name" placeholder="商品名称" />
            <input v-model="productForm.price" type="digit" placeholder="价格（元）" />
            <view class="merchant-upload-row">
              <input v-model="productForm.image_url" placeholder="商品图片 URL" />
              <button class="merchant-mini-btn" :loading="merchantLoading" @tap="chooseProductFormImage">上传</button>
            </view>
            <image v-if="productForm.image_url" class="merchant-image-preview" :src="productForm.image_url" mode="aspectFill" />
            <view class="merchant-form-actions">
              <button class="merchant-mini-btn" @tap="productFormOpen = false">取消</button>
              <button class="submit-order" :loading="merchantLoading" @tap="createProduct">保存商品</button>
            </view>
          </view>
        </view>
      </view>
    </view>

    <template v-else>
    <button v-if="hasMerchantEntry" class="identity-switch customer-side" @tap="switchAppMode(preferredMerchantMode)">{{ preferredMerchantMode === 'merchant' ? '商户端' : '申请页' }}</button>
    <view v-if="activeView === 'map'" class="map-screen">
      <map
        id="customerMap"
        class="native-map"
        :latitude="mapCenter.lat"
        :longitude="mapCenter.lng"
        :scale="focusedMerchantID ? 18 : 17"
        :markers="mapMarkers"
        show-location
        @markertap="onMarkerTap"
        @regionchange="onRegionChange"
      />

      <view class="customer-toolbar map-toolbar">
        <view class="toolbar-search">
          <input v-model="query" class="c-input" confirm-type="search" placeholder="搜索摊位 / 位置" @confirm="reloadCurrent" />
          <button v-if="query" class="search-clear" @tap="clearSearchQuery">×</button>
        </view>
        <button :class="['favorite-entry', favoriteEntryAnimating && 'is-opening']" @tap="openFavorites"><image class="toolbar-star-icon" :src="pinIconDataUri(false)" mode="aspectFit" /></button>
        <button class="view-toggle" @tap="switchView('list')"><image src="/static/icons/list.png" mode="aspectFit" /></button>
        <scroll-view class="category-strip" scroll-x>
          <view class="category-track">
            <button
              v-for="category in categories"
              :key="category"
              :class="['category-pill', selectedCategories.includes(category) && 'is-active']"
              @tap="toggleCategory(category)"
            >
              <view class="category-icon" :style="{ backgroundColor: categoryColor(category) }"><image :src="categoryIconDataUri(category)" mode="aspectFit" /></view>
              <text>{{ category }}</text>
            </button>
          </view>
        </scroll-view>
        <picker v-if="showDevLocation" class="dev-location-select" mode="selector" range-key="name" :range="devLocations" @change="chooseDevLocation">
          <view>开发定位</view>
        </picker>
      </view>

      <view v-if="locating" class="map-status">正在定位</view>
    </view>

    <view v-else class="vendor-list-page">
      <view class="customer-toolbar vendor-list-toolbar">
        <view class="toolbar-search">
          <input v-model="query" class="c-input" confirm-type="search" placeholder="搜索摊位 / 位置" @confirm="reloadCurrent" />
          <button v-if="query" class="search-clear" @tap="clearSearchQuery">×</button>
        </view>
        <button :class="['favorite-entry', favoriteEntryAnimating && 'is-opening']" @tap="openFavorites"><image class="toolbar-star-icon" :src="pinIconDataUri(false)" mode="aspectFit" /></button>
        <button class="view-toggle" @tap="switchView('map')"><image src="/static/icons/map-light.png" mode="aspectFit" /></button>
        <scroll-view class="category-strip" scroll-x>
          <view class="category-track">
            <button
              v-for="category in categories"
              :key="category"
              :class="['category-pill', selectedCategories.includes(category) && 'is-active']"
              @tap="toggleCategory(category)"
            >
              <view class="category-icon" :style="{ backgroundColor: categoryColor(category) }"><image :src="categoryIconDataUri(category)" mode="aspectFit" /></view>
              <text>{{ category }}</text>
            </button>
          </view>
        </scroll-view>
        <picker v-if="showDevLocation" class="dev-location-select" mode="selector" range-key="name" :range="devLocations" @change="chooseDevLocation">
          <view>开发定位</view>
        </picker>
      </view>

      <view v-if="locationNotice && !loading && !apiError" class="c-panel location-notice-card">
        <view>
          <text class="empty-title">未获取当前位置</text>
          <text class="empty-copy">{{ locationNotice }}</text>
        </view>
        <view class="location-notice-actions">
          <button class="c-btn primary" @tap="retryCustomerLocation">重新定位</button>
          <button class="c-btn secondary" @tap="openCustomerLocationSettings">去设置</button>
        </view>
      </view>
      <view v-if="apiError" class="c-panel empty-panel error-card">
        <text>{{ apiError }}</text>
        <button class="c-btn primary" @tap="reloadCurrent">重试</button>
      </view>
      <view v-else-if="loading" class="c-panel empty-panel">
        <text>正在加载附近摊主…</text>
      </view>
      <view v-else-if="!filteredVendors.length" class="c-panel empty-panel vendor-empty">
        <text class="empty-title">{{ query || selectedCategories.length ? '没有匹配摊主' : '附近暂无出摊摊主' }}</text>
        <text class="empty-copy">{{ query || selectedCategories.length ? '换个关键词或分类试试。' : '请稍后刷新' }}</text>
      </view>

      <view v-else class="vendor-card-list">
        <view v-for="vendor in displayedVendors" :key="vendor.id" class="c-panel vendor-card" @tap="openProducts(vendor)">
          <button :class="['favorite-star', isFavorite(vendor) && 'is-active', isFavoriteAnimating(vendor) && 'is-tapping']" @tap.stop="toggleFavorite(vendor)">
            <image class="favorite-star-icon" :src="pinIconDataUri(isFavorite(vendor))" mode="aspectFit" />
          </button>
          <view :class="['vendor-rank-photo', !stallVisual(vendor) && 'is-placeholder']">
            <image v-if="stallVisual(vendor)" :src="stallVisual(vendor)" mode="aspectFill" />
            <text v-else>{{ categoryInitial(vendor.category) }}</text>
          </view>
          <view class="vendor-card-main">
            <view class="vendor-title-block">
              <text class="vendor-title">{{ vendor.name }}</text>
              <view class="vendor-subtitle">
                <text>{{ vendor.category }}</text>
                <text>{{ distanceText(vendor, userLocation) }}</text>
                <text v-if="shouldShowEndText(vendor)">营业至 {{ vendor.endText }}</text>
              </view>
            </view>
            <text class="vendor-address">{{ vendor.address || vendor.area }}</text>
            <view class="vendor-products">
              <text v-for="product in previewProducts(vendor)" :key="product.id || product.name">{{ product.name }} {{ money(product.price_cents) }}</text>
              <text v-if="(vendor.products || []).length > 3">+{{ vendor.products.length - 3 }} 款</text>
              <text v-if="!previewProducts(vendor).length">今日商品待摊主补充</text>
            </view>
          </view>
        </view>
      </view>
    </view>


    <view v-if="productSheetOpen && productVendor" class="sheet-mask" @tap="closeProductSheet" @touchmove.stop.prevent>
      <view class="bottom-sheet product-sheet stall-products-sheet" @tap.stop @touchmove.stop>
        <button v-if="productSheetSource !== 'map'" class="modal-map-action" @tap="viewProductOnMap"><image src="/static/icons/map.png" mode="aspectFit" /></button>
        <button v-else class="modal-map-action modal-nav-action" @tap="navigateToProductVendor"><image :src="navigationIconDataUri()" mode="aspectFit" /></button>
        <button class="detail-close" @tap="closeProductSheet">×</button>
        <view :class="['stall-products-hero', !stallVisual(productVendor) && 'is-placeholder']">
          <image v-if="stallVisual(productVendor)" :src="stallVisual(productVendor)" mode="aspectFill" />
          <image v-else :src="categoryIconDataUri(productVendor.category)" mode="aspectFit" />
          <view class="stall-products-hero-info">
            <text class="stall-products-title">{{ productVendor.name }}</text>
            <view class="stall-products-meta-row">
              <view class="stall-products-meta-copy">
                <text>{{ productVendor.category }} · {{ productVendor.address || productVendor.area }}</text>
                <text v-if="productVendor.endText" class="stall-products-end">营业至 {{ productVendor.endText }}</text>
              </view>
              <button :class="['modal-favorite-star', isFavorite(productVendor) && 'is-active', isFavoriteAnimating(productVendor) && 'is-tapping']" @tap.stop="toggleFavorite(productVendor)">
                <image class="favorite-star-icon" :src="pinIconDataUri(isFavorite(productVendor))" mode="aspectFit" />
              </button>
            </view>
          </view>
        </view>
        <scroll-view class="stall-products-list-scroll" scroll-y enhanced :show-scrollbar="false" @touchmove.stop>
          <view v-if="productLoading" class="stall-products-list stall-products-skeleton">
            <view v-for="index in 3" :key="index" class="stall-product-row skeleton-product-row">
              <view class="stall-product-thumb skeleton-thumb"></view>
              <view class="stall-product-content">
                <view class="skeleton-line is-title"></view>
                <view class="skeleton-line is-desc"></view>
                <view class="skeleton-line is-price"></view>
              </view>
            </view>
          </view>
          <view v-else-if="!productVendor.products?.length" class="state-card empty-card"><text>摊主还没有补充今日商品。</text></view>
          <view v-else class="stall-products-list">
            <view v-for="product in productVendor.products" :key="product.id || product.name" class="stall-product-row">
              <view :class="['stall-product-thumb', productImage(product) && 'can-preview', !productImage(product) && 'is-placeholder']" @tap.stop="previewProductImage(product)">
                <image v-if="productImage(product)" :src="productImage(product)" mode="aspectFill" />
                <image v-else :src="categoryIconDataUri(productVendor.category)" mode="aspectFit" />
              </view>
              <view class="stall-product-content">
                <text class="product-name">{{ product.name }}</text>
                <text v-if="product.description" class="product-desc">{{ product.description }}</text>
                <text class="product-price">{{ money(product.price_cents) }}</text>
              </view>
            </view>
          </view>
        </scroll-view>
      </view>
    </view>

    <view v-if="productImagePreview.open" class="product-preview-mask" @tap="closeProductImagePreview">
      <view class="product-preview-dialog" @tap.stop>
        <button class="product-preview-close" @tap="closeProductImagePreview">×</button>
        <image class="product-preview-image" :src="productImagePreview.url" mode="widthFix" />
        <text v-if="productImagePreview.name" class="product-preview-title">{{ productImagePreview.name }}</text>
      </view>
    </view>

    <view v-if="favoritesOpen" class="sheet-mask" @tap="favoritesOpen = false">
      <view class="bottom-sheet favorites-sheet" @tap.stop>
        <button class="detail-close" @tap="favoritesOpen = false">×</button>
        <view class="favorites-search toolbar-search">
          <input v-model="favoritesQuery" class="c-input" confirm-type="search" placeholder="检索收藏摊位" @confirm="searchFavorites" />
          <button v-if="favoritesQuery" class="search-clear" @tap="clearFavoritesSearch">×</button>
        </view>
        <scroll-view class="favorites-list-scroll" scroll-y @scrolltolower="loadMoreFavorites">
          <view v-if="!filteredFavorites.length" class="state-card empty-card">
            <text>{{ favoritesQuery ? '没有匹配收藏，换个关键词试试。' : '暂无收藏，在摊位详情点收藏后会出现在这里。' }}</text>
          </view>
          <view v-else class="favorite-card-list">
            <view
              v-for="vendor in displayedFavorites"
              :key="vendor.id"
              :class="['favorite-swipe-row', canSwipeFavorite(vendor) && 'can-swipe', favoriteSwipeOpenId === vendor.id && 'is-open']"
              @touchstart="onFavoriteTouchStart(vendor, $event)"
              @touchend="onFavoriteTouchEnd(vendor, $event)"
              @touchcancel="onFavoriteTouchCancel"
            >
              <button v-if="canSwipeFavorite(vendor)" class="favorite-remove-action" @tap.stop="confirmRemoveFavorite(vendor)">移除</button>
              <view class="favorite-card" :style="favoriteCardStyle(vendor)" @tap="selectFavorite(vendor)">
                <view :class="['favorite-thumb', !stallVisual(vendor) && 'is-placeholder']">
                  <image v-if="stallVisual(vendor)" :src="stallVisual(vendor)" mode="aspectFill" />
                  <image v-else :src="categoryIconDataUri(vendor.category)" mode="aspectFit" />
                </view>
                <view class="favorite-card-main">
                  <view class="card-head">
                    <view class="favorite-title-block">
                      <text class="favorite-title">{{ vendor.name }} ›</text>
                      <text class="favorite-muted">{{ vendor.category }} · {{ vendor.address || vendor.area }}</text>
                    </view>
                    <text :class="['status-pill', vendor.isOpen && 'open']">{{ vendor.statusText }}</text>
                  </view>
                </view>
              </view>
            </view>
          </view>
          <view v-if="joinCtaVisible && !hasMerchantEntry" class="favorites-join-card">
            <view class="favorites-join-copy" @tap="openJoinFromFavorites">
              <text class="favorites-join-title">我是摊主，申请入驻</text>
              <text class="favorites-join-muted">登记摊位信息，让顾客在地图和列表里找到你。</text>
            </view>
            <button class="favorites-join-close" @tap.stop="dismissJoinCta">×</button>
          </view>
        </scroll-view>
      </view>
    </view>

    <view v-if="formSheet" class="sheet-mask" @tap="formSheet = ''">
      <view class="bottom-sheet" @tap.stop>
        <view class="sheet-handle"></view>
        <view class="sheet-head">
          <view>
            <text class="sheet-title">{{ formSheet === 'join' ? '申请入驻' : '反馈问题' }}</text>
            <text class="sheet-subtitle">平台会尽快处理你的信息</text>
          </view>
          <button class="close-button" @tap="formSheet = ''">关闭</button>
        </view>
        <view v-if="formSheet === 'join'" class="simple-form">
          <input v-model="joinForm.merchant_name" placeholder="摊位/商户名" />
          <input v-model="joinForm.contact_name" placeholder="联系人" />
          <input v-model="joinForm.contact_phone" type="number" placeholder="联系方式" />
          <picker mode="selector" :range="categories" :value="joinCategoryIndex" @change="chooseJoinCategory">
            <view :class="['form-picker', !joinForm.category && 'is-placeholder']">
              {{ joinForm.category || '请选择摊位类型' }}
            </view>
          </picker>
          <view class="merchant-upload-row">
            <input v-model="joinForm.photo_url" placeholder="摊铺图片 URL（可选）" />
            <button class="merchant-mini-btn" :loading="merchantLoading" @tap="chooseJoinPhoto">上传</button>
          </view>
          <image v-if="joinForm.photo_url" class="merchant-image-preview" :src="joinForm.photo_url" mode="aspectFill" />
          <input v-model="joinForm.usual_area" placeholder="常出摊区域" />
          <textarea v-model="joinForm.remark" placeholder="补充说明：营业时段、主卖商品等" />
          <button class="submit-order" :loading="submittingForm" @tap="submitJoin">提交申请</button>
        </view>
        <view v-else class="simple-form">
          <input v-model="feedbackForm.contact_name" placeholder="联系人" />
          <input v-model="feedbackForm.contact_phone" type="number" placeholder="联系方式" />
          <textarea v-model="feedbackForm.description" placeholder="请描述遇到的问题或建议" />
          <button class="submit-order" :loading="submittingForm" @tap="submitFeedback">提交反馈</button>
        </view>
      </view>
    </view>
    </template>

    <view v-if="sessionSheetOpen" class="sheet-mask" @tap="sessionSheetOpen = false">
      <view class="bottom-sheet" @tap.stop>
        <view class="sheet-handle"></view>
        <view class="sheet-head">
          <view>
            <text class="sheet-title">开始出摊</text>
            <text class="sheet-subtitle">定位成功后，顾客会在地图看到你的位置</text>
          </view>
          <button class="close-button" @tap="sessionSheetOpen = false">关闭</button>
        </view>
        <view class="simple-form merchant-simple-form">
          <picker mode="time" :value="sessionEndTime" @change="chooseSessionEndTime"><view class="form-picker">预计结束 {{ sessionEndTime }}</view></picker>
          <input v-model="sessionForm.address" placeholder="位置描述，例如 小区南门便利店旁" />
          <view class="merchant-upload-row">
            <input v-model="sessionForm.photo_url" placeholder="出摊照片 URL（可选）" />
            <button class="merchant-mini-btn" :loading="merchantLoading" @tap="chooseSessionPhoto">上传</button>
          </view>
          <image v-if="sessionForm.photo_url" class="merchant-image-preview" :src="sessionForm.photo_url" mode="aspectFill" />
          <view class="state-card"><text>{{ sessionLocationText }}</text></view>
          <button class="merchant-mini-btn" :loading="locating" @tap="locateForSession">重新定位</button>
          <button v-if="hasSessionLocation" class="submit-order" :loading="merchantSessionLoading" @tap="startMerchantSession">开始出摊</button>
        </view>
      </view>
    </view>
  </view>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { onLoad, onPullDownRefresh, onReachBottom, onShareAppMessage, onShareTimeline, onUnload } from '@dcloudio/uni-app'
import { customerApi, ensureCustomerLogin, normalizeMapItem } from '../../api/customer'
import { chooseAndUploadImage, getJSONStorage, money, setJSONStorage } from '../../api/client'
import { merchantApi, statusDescription as merchantStatusDescription, statusTitle as merchantStatusTitleText } from '../../api/merchant'
import { categoryColor, categoryIconDataUri, categoryInitial, STALL_CATEGORY_OPTIONS } from '../../utils/catalog'
import { distanceText, getUserLocation, hasFiniteLocation, openVendorLocation } from '../../utils/location'

const categories = STALL_CATEGORY_OPTIONS
const listPageSize = 12
const favoritePageSize = 12
const merchantProductPageSize = 12
const maxPinnedProducts = 3
const customerProductPreviewLimit = 3
const joinCtaDismissKey = 'mplzJoinCtaDismissedAt'
const joinCtaHiddenMs = 7 * 24 * 60 * 60 * 1000
const showDevLocation = import.meta.env.DEV
const devLocations = [
  { name: '旺角 E2 口', lat: 22.3193, lng: 114.1694, accuracy: 12 },
  { name: '创意园东门', lat: 22.3221, lng: 114.1712, accuracy: 18 },
  { name: '社区南门', lat: 22.3168, lng: 114.1678, accuracy: 15 },
  { name: '青榕广场', lat: 22.3242, lng: 114.1669, accuracy: 20 }
]
const loading = ref(false)
const locating = ref(false)
const productLoading = ref(false)
const submittingForm = ref(false)
const submittingApplication = ref(false)
const merchantLoading = ref(false)
const merchantSessionLoading = ref(false)
const appMode = ref('customer')
const activeView = ref('list')
const query = ref('')
const selectedCategories = ref([])
const vendors = ref([])
const listVisibleLimit = ref(listPageSize)
const favoriteItems = ref([])
const favoriteIds = ref(getJSONStorage('mplzFavoritesMp', []))
const favoriteAnimatingId = ref('')
const favoriteEntryAnimating = ref(false)
const selectedVendorId = ref('')
const selectedMarkerPulse = ref(false)
const focusedMerchantID = ref('')
const shareCode = ref('')
const userLocation = ref(null)
const viewportBounds = ref(null)
const viewportUserIntent = ref(false)
const apiError = ref('')
const locationNotice = ref('')
const productSheetOpen = ref(false)
const productVendorId = ref('')
const productSheetSource = ref('list')
const productImagePreview = reactive({ open: false, url: '', name: '' })
const favoritesOpen = ref(false)
const favoritesQuery = ref('')
const favoritesVisibleLimit = ref(favoritePageSize)
const favoriteSwipeOpenId = ref('')
const favoriteSwipeStart = ref({ id: '', x: 0, y: 0 })
const formSheet = ref('')
const joinCtaVisible = ref(shouldShowJoinCta())

const storedContact = getJSONStorage('mplzCustomerContact', {}) || {}
const joinForm = reactive({ merchant_name: '', contact_name: storedContact.name || '', contact_phone: storedContact.phone || '', category: '', photo_url: '', usual_area: '', remark: '' })
const feedbackForm = reactive({ contact_name: storedContact.name || '', contact_phone: storedContact.phone || '', description: '' })
const userPageState = ref({ page_mode: 'customer', next_action: 'create_application' })
const merchantStatus = ref(null)
const merchantDashboard = ref(null)
const merchantProducts = ref([])
const merchantProductsPage = ref(1)
const merchantProductsTotal = ref(0)
const merchantProductsLoading = ref(false)
const merchantProductsFinished = ref(false)
const merchantQrDataUrl = ref('')
const merchantShareCode = ref('')
const merchantTab = ref('overview')
const productFormOpen = ref(false)
const productEditing = ref({ id: '', field: '' })
const sessionSheetOpen = ref(false)
const sessionLocationText = ref('打开弹窗后会自动定位，请允许小程序获取位置。')
const productForm = reactive({ name: '', price: '', image_url: '' })
const merchantApplyForm = reactive({ merchant_name: '', contact_name: storedContact.name || '', contact_phone: storedContact.phone || '', category: '', photo_url: '', usual_area: '', remark: '' })
const sessionForm = reactive({ expected_end_at: '', address: '', lat: 0, lng: 0, accuracy: 0, photo_url: '' })
const navMetrics = ref({ titleTop: 18, titleHeight: 32, titleRight: 132, toolbarTop: 62 })
let viewportTimer = null
let selectedMarkerTimer = null
let locationRefreshPromise = null

const pageStyle = computed(() => `--nav-title-top:${navMetrics.value.titleTop}px;--nav-title-height:${navMetrics.value.titleHeight}px;--nav-title-right:${navMetrics.value.titleRight}px;--toolbar-top:${navMetrics.value.toolbarTop}px;`)

const filteredVendors = computed(() => {
  const keyword = query.value.trim().toLowerCase()
  return vendors.value
    .filter((vendor) => vendor.isOpen)
    .filter((vendor) => !keyword || [vendor.name, vendor.category, vendor.address, vendor.area].join(' ').toLowerCase().includes(keyword))
    .filter((vendor) => !selectedCategories.value.length || selectedCategories.value.includes(vendor.category))
    .sort((a, b) => comparableDistance(a) - comparableDistance(b))
})
const displayedVendors = computed(() => filteredVendors.value.slice(0, listVisibleLimit.value))

const favoriteVendors = computed(() => {
  if (favoritesQuery.value.trim()) return favoriteItems.value
  const byID = new Map(favoriteItems.value.map((vendor) => [vendor.id, vendor]))
  vendors.value.forEach((vendor) => { if (favoriteIds.value.includes(vendor.id)) byID.set(vendor.id, vendor) })
  return [...byID.values()]
})

const filteredFavorites = computed(() => favoriteVendors.value)
const displayedFavorites = computed(() => filteredFavorites.value.slice(0, favoritesVisibleLimit.value))
const selectedVendor = computed(() => [...vendors.value, ...favoriteItems.value].find((vendor) => vendor.id === selectedVendorId.value) || null)
const productVendor = computed(() => [...vendors.value, ...favoriteItems.value].find((vendor) => vendor.id === productVendorId.value) || null)
const joinCategoryIndex = computed(() => Math.max(0, categories.indexOf(joinForm.category)))
const merchantApplication = computed(() => merchantStatus.value?.application || null)
const currentIdentityState = computed(() => merchantStatus.value || userPageState.value || {})
const merchantStatusTitle = computed(() => merchantStatusTitleText(currentIdentityState.value?.next_action))
const merchantStatusCopy = computed(() => merchantStatusDescription(currentIdentityState.value || {}))
const merchantPinnedProductCount = computed(() => merchantProducts.value.filter((product) => hasPinnedSignal(product)).length)
const needsApplicationForm = computed(() => ['create_application', 'application_rejected'].includes(currentIdentityState.value?.next_action || 'create_application'))
const applicationStatusClass = computed(() => merchantApplication.value?.status === 'approved' ? 'green' : merchantApplication.value?.status === 'rejected' ? 'red' : '')
const applicationStatusText = computed(() => ({
  pending: '待审核',
  rejected: '未通过',
  approved: '已通过'
})[merchantApplication.value?.status] || (currentIdentityState.value?.next_action === 'create_application' ? '未提交' : '待处理'))
const merchantNextStepText = computed(() => currentIdentityState.value?.next_action === 'application_pending' ? '审核通过后再次打开小程序会进入商户工作台。' : currentIdentityState.value?.next_action === 'dashboard' ? '已通过，可进入商户工作台。' : '请根据审核说明处理。')
const merchantApplyCategoryIndex = computed(() => Math.max(0, categories.indexOf(merchantApplyForm.category)))
const hasMerchantEntry = computed(() => Boolean(merchantApplication.value || userPageState.value?.page_mode === 'application' || userPageState.value?.page_mode === 'merchant'))
const preferredMerchantMode = computed(() => userPageState.value?.page_mode === 'merchant' || currentIdentityState.value?.next_action === 'dashboard' ? 'merchant' : 'application')
const appTitle = computed(() => appMode.value === 'merchant' ? '商户工作台' : appMode.value === 'application' ? '入驻申请' : '出摊吧')
const merchantDisplayName = computed(() => merchantDashboard.value?.merchant?.display_name || merchantStatus.value?.merchant?.display_name || merchantApplication.value?.merchant_name || '我的摊位')
const applicationInfoRows = computed(() => {
  const app = merchantApplication.value || {}
  return [
    { label: '摊位名称', value: app.merchant_name || '未填写' },
    { label: '摊位类型', value: app.category || '未选择' },
    { label: '联系人', value: app.contact_name || '未填写' },
    { label: '联系电话', value: app.contact_phone || '未填写' },
    { label: '常出摊区域', value: app.usual_area || '未填写' },
    { label: '补充说明', value: app.remark || '暂无' }
  ]
})
const activeMerchantSession = computed(() => {
  const session = merchantDashboard.value?.stall_session
  if (!session || session.status !== 'active') return null
  if (session.expected_end_at && new Date(session.expected_end_at).getTime() <= Date.now()) return null
  return session
})
const merchantSessionAddress = computed(() => cleanManualAddress(activeMerchantSession.value?.address))
const sessionEndTime = computed(() => sessionForm.expected_end_at ? sessionForm.expected_end_at.slice(11, 16) : localTimeValue(new Date(Date.now() + 4 * 3600 * 1000)))
const hasSessionLocation = computed(() => hasFiniteLocation(sessionForm))
const mapCenter = computed(() => {
  if (selectedVendor.value && hasFiniteLocation(selectedVendor.value)) return { lat: selectedVendor.value.lat, lng: selectedVendor.value.lng }
  const valid = filteredVendors.value.filter(hasFiniteLocation)
  if (valid.length) return {
    lat: valid.reduce((sum, vendor) => sum + Number(vendor.lat), 0) / valid.length,
    lng: valid.reduce((sum, vendor) => sum + Number(vendor.lng), 0) / valid.length
  }
  if (userLocation.value && hasFiniteLocation(userLocation.value)) return userLocation.value
  return { lat: 22.3193, lng: 114.1694 }
})
const mapMarkers = computed(() => filteredVendors.value.filter(hasFiniteLocation).map((vendor, index) => ({
  id: index + 1,
  vendorId: vendor.id,
  latitude: Number(vendor.lat),
  longitude: Number(vendor.lng),
  iconPath: markerIconPath(vendor.category),
  width: markerSize(vendor).width,
  height: markerSize(vendor).height,
  anchor: { x: 0.5, y: 1 },
  zIndex: selectedVendorId.value === vendor.id ? 10 : 1,
  callout: {
    content: vendor.name,
    color: '#2f1f0d',
    fontSize: 11,
    borderRadius: 8,
    bgColor: '#FFF7E8',
    padding: 6,
    display: 'ALWAYS'
  }
})))

function markerSize(vendor) {
  if (selectedVendorId.value !== vendor.id) return { width: 30, height: 36 }
  return selectedMarkerPulse.value ? { width: 44, height: 52 } : { width: 36, height: 42 }
}

onLoad(async (options = {}) => {
  applyNavigationMetrics()
  joinCtaVisible.value = shouldShowJoinCta()
  shareCode.value = String(options.shareCode || options.scene || '')
  focusedMerchantID.value = String(options.merchantId || options.merchant_id || '')
  if (shareCode.value || focusedMerchantID.value) activeView.value = 'map'
  await bootstrap()
})

onPullDownRefresh(async () => {
  await refreshCurrentMode()
  uni.stopPullDownRefresh()
})

onReachBottom(() => {
  if (appMode.value === 'customer' && activeView.value === 'list') loadMoreVendors()
  if (appMode.value === 'merchant' && merchantTab.value === 'products') loadProducts()
})

onUnload(() => {
  clearTimeout(viewportTimer)
  clearSelectedMarkerPulse()
})

onShareAppMessage(() => ({ title: shareTitle(), path: sharePath() }))
onShareTimeline(() => ({ title: shareTitle(), query: shareQuery() }))

async function bootstrap() {
  applyStartupDevLocation()
  const loggedIn = await ensureCustomerLogin().catch((error) => {
    apiError.value = error.message || '小程序登录失败'
    uni.showToast({ title: apiError.value, icon: 'none' })
    return false
  })
  if (!loggedIn) {
    vendors.value = []
    favoriteItems.value = []
    favoriteIds.value = []
    return
  }
  await resolveShare()
  await loadUserPageState({ silent: true }).catch(() => {})
  applyInitialAppMode()
  if (appMode.value === 'merchant') {
    await loadMerchantHome().catch((error) => uni.showToast({ title: error.message || '商户数据加载失败', icon: 'none' }))
    return
  }
  if (appMode.value === 'application') {
    await loadMerchantStatus({ silent: true }).catch((error) => uni.showToast({ title: error.message || '申请状态加载失败', icon: 'none' }))
    return
  }
  await loadVendors()
}

async function refreshCurrentMode() {
  await loadUserPageState({ silent: true }).catch(() => {})
  if (appMode.value === 'merchant') {
    if (userPageState.value?.page_mode !== 'merchant') {
      applyInitialAppMode()
      if (appMode.value === 'customer') await loadVendors()
      return
    }
    if (merchantTab.value === 'products') await loadProducts({ reset: true }).catch((error) => uni.showToast({ title: error.message || '商品加载失败', icon: 'none' }))
    else await loadMerchantHome().catch((error) => uni.showToast({ title: error.message || '商户数据加载失败', icon: 'none' }))
    return
  }
  if (appMode.value === 'application') {
    if (userPageState.value?.page_mode === 'merchant') {
      appMode.value = 'merchant'
      await loadMerchantHome().catch((error) => uni.showToast({ title: error.message || '商户数据加载失败', icon: 'none' }))
      return
    }
    if (userPageState.value?.page_mode === 'application') await loadMerchantStatus({ silent: true }).catch((error) => uni.showToast({ title: error.message || '申请状态加载失败', icon: 'none' }))
    else {
      appMode.value = 'customer'
      await loadVendors()
    }
    return
  }
  await loadVendors()
}

function applyNavigationMetrics() {
  try {
    const menu = uni.getMenuButtonBoundingClientRect?.()
    const windowInfo = uni.getWindowInfo?.()
    if (!menu?.top || !menu?.bottom) return
    const windowWidth = Number(windowInfo?.windowWidth || 390)
    navMetrics.value = {
      titleTop: Math.max(12, Number(menu.top)),
      titleHeight: Number(menu.height || 32),
      titleRight: Math.max(120, windowWidth - Number(menu.left || windowWidth) + 12),
      toolbarTop: Number(menu.bottom) + 12
    }
  } catch {}
}

function applyStartupDevLocation() {
  if (!showDevLocation || userLocation.value) return
  userLocation.value = { ...devLocations[0] }
}

async function refreshLocation(options = {}) {
  if (showDevLocation && options.silent) {
    applyStartupDevLocation()
    locationNotice.value = ''
    return true
  }
  if (!options.silent) locating.value = true
  try {
    userLocation.value = await getUserLocation()
    locationNotice.value = ''
    return true
  } catch (error) {
    locationNotice.value = locationFailureText(error)
    if (!options.silent) uni.showToast({ title: '定位失败，可继续浏览', icon: 'none' })
    return false
  } finally {
    locating.value = false
  }
}

function locationFailureText(error) {
  const message = String(error?.message || error?.errMsg || '')
  if (/auth deny|authorize|permission|denied|拒绝|未授权/i.test(message)) return '你可能拒绝了位置授权。开启定位后，可以按距离查看附近出摊摊主。'
  if (/system|location service|定位服务|service/i.test(message)) return '手机系统或微信定位权限可能未开启。开启后可以按距离展示附近摊主。'
  return '暂时无法获取手机位置，当前按默认范围展示摊主。开启定位后距离和附近排序会更准确。'
}

async function resolveShare() {
  if (!shareCode.value || focusedMerchantID.value) return
  try {
    const data = await customerApi.getShare(shareCode.value)
    const merchant = data?.merchant || data
    focusedMerchantID.value = data?.share?.merchant_id || merchant?.id || data?.merchant_id || ''
  } catch (error) {
    apiError.value = error.message || '分享链接无效'
  }
}

function locationParams() {
  return userLocation.value ? { lat: userLocation.value.lat, lng: userLocation.value.lng } : {}
}

async function loadVendors(options = {}) {
  if (!options.background) loading.value = true
  apiError.value = ''
  try {
    if (!userLocation.value && !options.skipLocation && !focusedMerchantID.value) {
      await ensureCustomerLocation({ silent: Boolean(options.background) })
    }
    if (focusedMerchantID.value) {
      const vendor = await customerApi.getMerchantMapState(focusedMerchantID.value, locationParams())
      vendors.value = vendor?.merchantId ? [vendor] : []
      selectedVendorId.value = vendors.value[0]?.id || ''
      if (vendors.value[0]) await ensureShareFavorite(vendors.value[0])
    } else {
      const mapBounds = activeView.value === 'map' && !query.value ? viewportBounds.value || {} : {}
      const resp = await customerApi.nearbyStalls({ ...locationParams(), ...mapBounds, q: query.value, categories: selectedCategories.value, limit: 20 })
      vendors.value = resp.data || []
      if (!vendors.value.some((vendor) => vendor.id === selectedVendorId.value)) selectedVendorId.value = ''
    }
    resetVendorPagination()
  } catch (error) {
    apiError.value = error.message || '摊位数据加载失败'
    uni.showToast({ title: apiError.value, icon: 'none' })
  } finally {
    if (!options.background) loading.value = false
  }
}

async function ensureCustomerLocation(options = {}) {
  if (userLocation.value) return true
  if (locationRefreshPromise) return locationRefreshPromise
  locationRefreshPromise = refreshLocation({ silent: options.silent }).finally(() => {
    locationRefreshPromise = null
  })
  return locationRefreshPromise
}

async function loadFavorites(options = {}) {
  const name = options.name ?? favoritesQuery.value
  const hasNameFilter = Boolean(String(name || '').trim())
  const resp = await customerApi.listFavorites({ page: 1, size: 15, name })
  favoriteItems.value = resp.data || []
  if (hasNameFilter) {
    const next = new Set(favoriteIds.value)
    favoriteItems.value.forEach((vendor) => next.add(vendor.id))
    favoriteIds.value = [...next]
  } else {
    favoriteIds.value = favoriteItems.value.map((vendor) => vendor.id)
  }
  if (!favoriteItems.value.some((vendor) => vendor.id === favoriteSwipeOpenId.value)) favoriteSwipeOpenId.value = ''
  saveFavorites()
}

async function loadUserPageState(options = {}) {
  try {
    userPageState.value = await customerApi.getPageState()
    if (hasMerchantEntry.value) joinCtaVisible.value = false
    if (options.showToast) uni.showToast({ title: '身份已刷新', icon: 'none' })
    return userPageState.value
  } catch (error) {
    if (!options.silent) uni.showToast({ title: error.message || '用户状态加载失败', icon: 'none' })
    throw error
  }
}

async function loadMerchantStatus(options = {}) {
  try {
    merchantStatus.value = await merchantApi.getApplicationStatus()
    syncUserPageStateFromMerchantStatus(merchantStatus.value)
    fillMerchantApplyForm(merchantStatus.value?.application || {})
    if (hasMerchantEntry.value) joinCtaVisible.value = false
    if (options.showToast) uni.showToast({ title: '状态已刷新', icon: 'none' })
    return merchantStatus.value
  } catch (error) {
    if (!options.silent) uni.showToast({ title: error.message || '申请状态加载失败', icon: 'none' })
    throw error
  }
}

function syncUserPageStateFromMerchantStatus(status = {}) {
  if (!status?.next_action) return
  const pageMode = status.next_action === 'dashboard' ? 'merchant' : status.application ? 'application' : 'customer'
  userPageState.value = {
    ...userPageState.value,
    page_mode: pageMode,
    next_action: status.next_action,
    is_merchant: pageMode === 'merchant',
    has_application: Boolean(status.application),
    application_id: status.application?.id || userPageState.value?.application_id || '',
    application_status: status.application?.status || userPageState.value?.application_status || '',
    merchant_id: status.merchant?.id || status.application?.merchant_id || userPageState.value?.merchant_id || null
  }
}

function applyInitialAppMode() {
  if (shareCode.value || focusedMerchantID.value) {
    appMode.value = 'customer'
    return
  }
  if (userPageState.value?.page_mode === 'merchant') {
    appMode.value = 'merchant'
    return
  }
  if (userPageState.value?.page_mode === 'application') {
    appMode.value = 'application'
    return
  }
  appMode.value = 'customer'
}

function fillMerchantApplyForm(app = {}) {
  merchantApplyForm.merchant_name = app.merchant_name || merchantApplyForm.merchant_name || joinForm.merchant_name
  merchantApplyForm.contact_name = app.contact_name || merchantApplyForm.contact_name || joinForm.contact_name
  merchantApplyForm.contact_phone = app.contact_phone || merchantApplyForm.contact_phone || joinForm.contact_phone
  merchantApplyForm.category = app.category || merchantApplyForm.category || joinForm.category
  merchantApplyForm.photo_url = app.photo_url || merchantApplyForm.photo_url || joinForm.photo_url || ''
  merchantApplyForm.usual_area = app.usual_area || merchantApplyForm.usual_area || joinForm.usual_area
  merchantApplyForm.remark = app.remark || merchantApplyForm.remark || joinForm.remark
}

async function ensureShareFavorite(vendor) {
  if (!vendor?.merchantId || favoriteIds.value.includes(vendor.id)) return
  try { await addFavorite(vendor, { silent: true }) } catch {}
}

function reloadCurrent() { loadVendors() }
function switchView(view) {
  activeView.value = view
  if (view === 'map') loadVendors()
}
function clearSearchQuery() {
  query.value = ''
  selectedVendorId.value = ''
  resetVendorPagination()
  loadVendors()
}
function chooseDevLocation(event) {
  const index = Number(event?.detail?.value)
  const location = devLocations[index]
  if (!location) return
  userLocation.value = { ...location }
  viewportBounds.value = null
  viewportUserIntent.value = false
  uni.showToast({ title: `已切换定位：${location.name}`, icon: 'none' })
  loadVendors()
}
function toggleCategory(category) {
  selectedVendorId.value = ''
  selectedCategories.value = selectedCategories.value.includes(category)
    ? selectedCategories.value.filter((item) => item !== category)
    : [...selectedCategories.value, category]
  viewportBounds.value = null
  resetVendorPagination()
  loadVendors()
}
function resetVendorPagination() {
  listVisibleLimit.value = listPageSize
}
function loadMoreVendors() {
  listVisibleLimit.value = Math.min(filteredVendors.value.length, listVisibleLimit.value + listPageSize)
}

async function retryCustomerLocation() {
  const ok = await refreshLocation()
  if (ok) await loadVendors({ skipLocation: true })
}

function openCustomerLocationSettings() {
  if (typeof uni.openSetting !== 'function') {
    uni.showToast({ title: '请在微信设置中开启位置权限', icon: 'none' })
    return
  }
  uni.openSetting({
    success: () => retryCustomerLocation(),
    fail: () => uni.showToast({ title: '请在微信设置中开启位置权限', icon: 'none' })
  })
}

function comparableDistance(vendor) {
  const meters = Number(vendor?.distanceMeters)
  if (Number.isFinite(meters)) return meters
  return Number.MAX_SAFE_INTEGER
}

function onMarkerTap(event) {
  const marker = mapMarkers.value.find((item) => Number(item.id) === Number(event.detail?.markerId))
  const vendor = marker ? filteredVendors.value.find((item) => item.id === marker.vendorId) : null
  if (!vendor) return
  openProducts(vendor)
}

function onRegionChange(event) {
  if (focusedMerchantID.value || activeView.value !== 'map') return
  const causedBy = event?.detail?.causedBy
  if (event?.type === 'begin' && causedBy !== 'update') viewportUserIntent.value = true
  if (event?.type === 'end' && viewportUserIntent.value) {
    viewportUserIntent.value = false
    scheduleViewportLoad()
  }
}

function scheduleViewportLoad() {
  clearTimeout(viewportTimer)
  viewportTimer = setTimeout(() => {
    const mapContext = uni.createMapContext('customerMap')
    mapContext?.getRegion?.({
      success: (region) => {
        const sw = region?.southwest
        const ne = region?.northeast
        if (!sw || !ne) return
        viewportBounds.value = {
          min_lat: sw.latitude,
          max_lat: ne.latitude,
          min_lng: sw.longitude,
          max_lng: ne.longitude
        }
        loadVendors()
      }
    })
  }, 450)
}

function isFavorite(vendor) { return favoriteIds.value.includes(vendor?.id) }
function isFavoriteAnimating(vendor) { return favoriteAnimatingId.value === String(vendor?.id || '') }
function triggerFavoriteMotion(vendor) {
  const id = String(vendor?.id || '')
  if (!id) return
  favoriteAnimatingId.value = ''
  setTimeout(() => { favoriteAnimatingId.value = id }, 0)
  setTimeout(() => {
    if (favoriteAnimatingId.value === id) favoriteAnimatingId.value = ''
  }, 420)
}
function saveFavorites() { setJSONStorage('mplzFavoritesMp', favoriteIds.value) }
async function toggleFavorite(vendor) {
  if (!vendor) return
  triggerFavoriteMotion(vendor)
  if (isFavorite(vendor)) return removeFavorite(vendor)
  return addFavorite(vendor)
}
async function addFavorite(vendor, options = {}) {
  if (!vendor?.merchantId) {
    if (!options.silent) uni.showToast({ title: '缺少商户信息', icon: 'none' })
    return
  }
  await customerApi.addFavorite({ merchant_id: vendor.merchantId })
  if (!favoriteIds.value.includes(vendor.id)) favoriteIds.value = [vendor.id, ...favoriteIds.value]
  if (!favoriteItems.value.some((item) => item.id === vendor.id)) favoriteItems.value = [vendor, ...favoriteItems.value]
  saveFavorites()
  if (!options.silent) uni.showToast({ title: '已收藏', icon: 'success' })
}
async function removeFavorite(vendor) {
  const favoriteId = vendor.favoriteId || favoriteItems.value.find((item) => item.id === vendor.id)?.favoriteId || await lookupFavoriteId(vendor)
  if (!favoriteId) {
    uni.showToast({ title: '未找到收藏记录，请刷新后重试', icon: 'none' })
    return
  }
  await customerApi.removeFavorite(favoriteId)
  favoriteIds.value = favoriteIds.value.filter((id) => id !== vendor.id)
  favoriteItems.value = favoriteItems.value.filter((item) => item.id !== vendor.id)
  if (favoriteSwipeOpenId.value === vendor.id) favoriteSwipeOpenId.value = ''
  saveFavorites()
  uni.showToast({ title: '已取消收藏', icon: 'none' })
}
async function lookupFavoriteId(vendor) {
  const resp = await customerApi.listFavorites({ merchant_id: vendor.merchantId })
  const item = (resp.data || []).find((row) => row.merchantId === vendor.merchantId)
  return item?.favoriteId || ''
}

async function openFavorites() {
  triggerFavoriteEntryMotion()
  const loggedIn = await ensureCustomerLogin().catch((error) => {
    uni.showToast({ title: error.message || '登录失败', icon: 'none' })
    return false
  })
  if (!loggedIn) return
  await loadFavorites().catch((error) => uni.showToast({ title: error.message || '收藏加载失败', icon: 'none' }))
  favoritesVisibleLimit.value = favoritePageSize
  favoriteSwipeOpenId.value = ''
  favoritesOpen.value = true
}
function triggerFavoriteEntryMotion() {
  favoriteEntryAnimating.value = false
  setTimeout(() => { favoriteEntryAnimating.value = true }, 0)
  setTimeout(() => { favoriteEntryAnimating.value = false }, 520)
}
async function searchFavorites() {
  favoritesVisibleLimit.value = favoritePageSize
  favoriteSwipeOpenId.value = ''
  await loadFavorites({ name: favoritesQuery.value }).catch((error) => uni.showToast({ title: error.message || '收藏查询失败', icon: 'none' }))
}
async function clearFavoritesSearch() {
  favoritesQuery.value = ''
  await searchFavorites()
}
function selectFavorite(vendor) {
  if (favoriteSwipeOpenId.value) {
    favoriteSwipeOpenId.value = ''
    return
  }
  if (!vendor.isOpen) {
    uni.showToast({ title: '摊主未出摊，暂无可跳转位置', icon: 'none' })
    return
  }
  if (!vendors.value.some((item) => item.id === vendor.id)) vendors.value = [vendor, ...vendors.value]
  favoritesOpen.value = false
  selectedVendorId.value = vendor.id
  activeView.value = 'map'
  startSelectedMarkerPulse()
}
function loadMoreFavorites() {
  favoritesVisibleLimit.value = Math.min(filteredFavorites.value.length, favoritesVisibleLimit.value + favoritePageSize)
}

function canSwipeFavorite(vendor) {
  return Boolean(vendor && !vendor.isOpen)
}

function favoriteCardStyle(vendor) {
  return favoriteSwipeOpenId.value === vendor?.id ? 'transform: translateX(-152rpx);' : ''
}

function onFavoriteTouchStart(vendor, event) {
  if (!canSwipeFavorite(vendor)) return
  const touch = event?.touches?.[0]
  if (!touch) return
  favoriteSwipeStart.value = { id: vendor.id, x: Number(touch.clientX || 0), y: Number(touch.clientY || 0) }
}

function onFavoriteTouchEnd(vendor, event) {
  const start = favoriteSwipeStart.value
  if (!canSwipeFavorite(vendor) || start.id !== vendor.id) return
  const touch = event?.changedTouches?.[0]
  favoriteSwipeStart.value = { id: '', x: 0, y: 0 }
  if (!touch) return
  const dx = Number(touch.clientX || 0) - start.x
  const dy = Number(touch.clientY || 0) - start.y
  if (Math.abs(dx) < 36 || Math.abs(dx) < Math.abs(dy) * 1.15) return
  favoriteSwipeOpenId.value = dx < 0 ? vendor.id : ''
}

function onFavoriteTouchCancel() {
  favoriteSwipeStart.value = { id: '', x: 0, y: 0 }
}

function confirmRemoveFavorite(vendor) {
  if (!canSwipeFavorite(vendor)) return
  uni.showModal({
    title: '移除收藏',
    content: `从收藏中移除「${vendor.name || '该摊主'}」？`,
    confirmText: '移除',
    confirmColor: '#B45309',
    success: async (res) => {
      if (!res.confirm) return
      await removeFavorite(vendor).catch((error) => uni.showToast({ title: error.message || '移除失败', icon: 'none' }))
    }
  })
}

async function openProducts(vendor) {
  productSheetSource.value = activeView.value
  productVendorId.value = vendor.id
  productSheetOpen.value = true
  if (!vendor.merchantId) {
    uni.showToast({ title: '缺少商户信息，暂不能加载商品', icon: 'none' })
    return
  }
  productLoading.value = true
  try {
    const productsResp = await customerApi.listProducts(vendor.merchantId, { page: 1, size: 10 })
    const nextVendor = {
      ...vendor,
      products: productsResp.data || [],
      productsTotal: productsResp.total ?? (productsResp.data || []).length,
      productsComplete: true
    }
    updateVendorCache(vendor.id, nextVendor)
    productVendorId.value = nextVendor.id
  } catch (error) {
    updateVendorCache(vendor.id, { ...vendor, products: [], productsComplete: false })
    uni.showToast({ title: error.message || '商品加载失败', icon: 'none' })
  } finally {
    productLoading.value = false
  }
}
function closeProductSheet() {
  closeProductImagePreview()
  productSheetOpen.value = false
  productVendorId.value = ''
}
function updateVendorCache(id, vendor) {
  vendors.value = vendors.value.map((item) => item.id === id ? vendor : item)
  favoriteItems.value = favoriteItems.value.map((item) => item.id === id ? vendor : item)
}
function stallVisual(vendor) { return vendor?.photoUrl || vendor?.avatarUrl || '' }
function previewProducts(vendor) { return (vendor.products || []).slice(0, customerProductPreviewLimit) }
function productImage(product) { return product?.image_url || product?.imageUrl || product?.image || '' }
function previewProductImage(product) {
  const current = productImage(product)
  if (!current) return
  productImagePreview.url = current
  productImagePreview.name = product?.name || ''
  productImagePreview.open = true
}
function closeProductImagePreview() {
  productImagePreview.open = false
  productImagePreview.url = ''
  productImagePreview.name = ''
}
function shouldShowEndText(vendor) {
  return Boolean(vendor?.endText) && !distanceText(vendor, userLocation.value).includes('营业至')
}
function viewProductOnMap() {
  if (!productVendor.value) return
  selectedVendorId.value = productVendor.value.id
  activeView.value = 'map'
  closeProductSheet()
  startSelectedMarkerPulse()
}

function navigateToProductVendor() {
  if (!productVendor.value) return
  openVendorLocation(productVendor.value)
}

function navigationIconDataUri() {
  const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100" fill="none"><path d="M78 16 43 84 36 52 16 43 78 16Z" fill="#2f1f0d" stroke="#2f1f0d" stroke-width="7" stroke-linejoin="round"/><path d="M43 84 49 55 78 16" stroke="#fffaf0" stroke-width="6" stroke-linecap="round" stroke-linejoin="round" opacity=".9"/></svg>'
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}

function markerIconPath(category = '') {
  const label = String(category || '')
  if (label.includes('早餐') || label.includes('小吃')) return '/static/markers/breakfast.png'
  if (label.includes('咖啡') || label.includes('饮品')) return '/static/markers/coffee.png'
  if (label.includes('水果') || label.includes('鲜切')) return '/static/markers/fruit.png'
  if (label.includes('夜宵') || label.includes('烧烤')) return '/static/markers/barbecue.png'
  if (label.includes('便当') || label.includes('快餐')) return '/static/markers/bento.png'
  if (label.includes('甜品') || label.includes('冷饮')) return '/static/markers/dessert.png'
  if (label.includes('卤味') || label.includes('熟食')) return '/static/markers/braise.png'
  if (label.includes('鲜花') || label.includes('手作')) return '/static/markers/flower.png'
  if (label.includes('生鲜') || label.includes('菜摊')) return '/static/markers/vegetable.png'
  return '/static/markers/other.png'
}
function startSelectedMarkerPulse() {
  clearSelectedMarkerPulse()
  selectedMarkerPulse.value = true
  let ticks = 0
  selectedMarkerTimer = setInterval(() => {
    selectedMarkerPulse.value = !selectedMarkerPulse.value
    ticks += 1
    if (ticks >= 8) {
      clearSelectedMarkerPulse()
      selectedMarkerPulse.value = false
    }
  }, 260)
}
function clearSelectedMarkerPulse() {
  if (selectedMarkerTimer) clearInterval(selectedMarkerTimer)
  selectedMarkerTimer = null
}
function shouldShowJoinCta() {
  const dismissedAt = Number(getJSONStorage(joinCtaDismissKey, 0) || 0)
  return !dismissedAt || Date.now() - dismissedAt >= joinCtaHiddenMs
}
function dismissJoinCta() {
  joinCtaVisible.value = false
  setJSONStorage(joinCtaDismissKey, Date.now())
}
function chooseJoinCategory(event) {
  const index = Number(event?.detail?.value)
  joinForm.category = categories[index] || ''
}
function chooseMerchantApplyCategory(event) {
  const index = Number(event?.detail?.value)
  merchantApplyForm.category = categories[index] || ''
}
function openJoinFromFavorites() {
  favoritesOpen.value = false
  openJoinSheet()
}
function openJoinSheet() { formSheet.value = 'join' }
function openFeedbackSheet() { formSheet.value = 'feedback' }

async function switchAppMode(mode) {
  if (mode === 'merchant' || mode === 'application') {
    await loadUserPageState({ silent: true }).catch(() => {})
    mode = preferredMerchantMode.value
    if (mode === 'application') await loadMerchantStatus({ silent: true }).catch(() => {})
  }
  appMode.value = mode
  if (mode === 'merchant') await loadMerchantHome().catch((error) => uni.showToast({ title: error.message || '商户数据加载失败', icon: 'none' }))
  if (mode === 'customer' && !vendors.value.length) await loadVendors()
}

async function submitApplication() {
  if (!merchantApplyForm.merchant_name || !merchantApplyForm.contact_phone || !merchantApplyForm.category) {
    uni.showToast({ title: '请填写摊位名、联系方式和类型', icon: 'none' })
    return
  }
  submittingApplication.value = true
  try {
    if (merchantApplication.value?.id) await merchantApi.updateApplication(merchantApplication.value.id, merchantApplyForm)
    else await customerApi.createApplication({ ...merchantApplyForm })
    await loadUserPageState({ silent: true }).catch(() => {})
    await loadMerchantStatus({ silent: true })
    appMode.value = userPageState.value?.page_mode === 'merchant' ? 'merchant' : 'application'
    if (appMode.value === 'merchant') await loadMerchantHome()
    uni.showToast({ title: '申请已提交', icon: 'success' })
  } catch (error) {
    uni.showToast({ title: error.message || '提交失败', icon: 'none' })
  } finally {
    submittingApplication.value = false
  }
}

async function loadMerchantHome() {
  const dashboard = await merchantApi.getDashboard()
  const sessions = await merchantApi.listStallSessions({ status: 'active', page: 1, size: 1 }).catch(() => ({}))
  merchantDashboard.value = { ...dashboard, stall_session: firstActiveSession(sessions) || activeSessionCandidate(dashboard.stall_session) }
  const merchant = merchantDashboard.value.merchant || {}
  merchantQrDataUrl.value = merchant.share_poster_url || merchant.share_qrcode_url || ''
  merchantShareCode.value = merchant.share_code || ''
}

function previewMerchantQr() {
  const url = merchantQrDataUrl.value
  if (!url) {
    uni.showToast({ title: '二维码生成中', icon: 'none' })
    return
  }
  uni.previewImage({ current: url, urls: [url] })
}

function firstActiveSession(resp) {
  if (activeSessionCandidate(resp)) return resp
  const sessions = Array.isArray(resp?.data) ? resp.data : []
  return sessions.find(activeSessionCandidate) || null
}

function activeSessionCandidate(session) {
  if (!session || session.status !== 'active') return null
  if (session.expected_end_at && new Date(session.expected_end_at).getTime() <= Date.now()) return null
  return session
}

async function switchMerchantTab(tab) {
  if (merchantTab.value === tab && tab !== 'overview') return
  merchantTab.value = tab
  if (tab === 'overview') await loadMerchantHome().catch((error) => uni.showToast({ title: error.message || '加载失败', icon: 'none' }))
  if (tab === 'products' && !merchantProducts.length) await loadProducts({ reset: true }).catch((error) => uni.showToast({ title: error.message || '商品加载失败', icon: 'none' }))
}

async function loadProducts(options = {}) {
  if (merchantProductsLoading.value) return
  const reset = Boolean(options.reset)
  if (!reset && merchantProductsFinished.value) return
  const page = reset ? 1 : merchantProductsPage.value
  merchantProductsLoading.value = true
  try {
    const resp = await merchantApi.listProducts({ page, size: merchantProductPageSize })
    const rows = resp.data || []
    const nextRows = reset ? rows : [...merchantProducts.value, ...rows]
    merchantProducts.value = nextRows
    merchantProductsTotal.value = Number(resp.total || 0)
    merchantProductsPage.value = page + 1
    merchantProductsFinished.value = rows.length < merchantProductPageSize || (merchantProductsTotal.value > 0 && merchantProducts.value.length >= merchantProductsTotal.value)
  } finally {
    merchantProductsLoading.value = false
  }
}

function productPriceYuan(product) {
  return (Number(product?.price_cents || 0) / 100).toFixed(2)
}
function productInitial(product) {
  return String(product?.name || '商').slice(0, 1)
}
function pinIconDataUri(active = false) {
  const fill = active ? '#f5a50a' : 'none'
  const stroke = active ? '#b7790d' : '#9a650a'
  const path = 'M50 7.5 62.7 33.2l28.4 4.1-20.5 20 4.8 28.3L50 72.2 24.6 85.6l4.8-28.3-20.5-20 28.4-4.1L50 7.5Z'
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="100" height="100" viewBox="0 0 100 100"><path fill="${fill}" stroke="${stroke}" stroke-width="8" stroke-linejoin="round" d="${path}"/></svg>`
  return `data:image/svg+xml;charset=UTF-8,${encodeURIComponent(svg)}`
}

async function createProduct() {
  if (!productForm.name || !productForm.price) return uni.showToast({ title: '请填写商品名称和价格', icon: 'none' })
  if (!productForm.image_url) return uni.showToast({ title: '请上传或填写商品图片', icon: 'none' })
  merchantLoading.value = true
  try {
    await merchantApi.createProduct({
      name: productForm.name,
      price_cents: Math.round(Number(productForm.price || 0) * 100),
      image_url: productForm.image_url,
      stock: 9999,
      status: 'on_sale'
    })
    Object.assign(productForm, { name: '', price: '', image_url: '' })
    productFormOpen.value = false
    await loadProducts({ reset: true })
    uni.showToast({ title: '商品已添加', icon: 'success' })
  } catch (error) {
    uni.showToast({ title: error.message || '保存失败', icon: 'none' })
  } finally {
    merchantLoading.value = false
  }
}

async function updateProduct(product) {
  if (!product?.id || !product.name) return
  await merchantApi.updateProduct(product).catch((error) => uni.showToast({ title: error.message || '商品更新失败', icon: 'none' }))
}

function isProductEditing(product, field) {
  return String(productEditing.value.id) === String(product?.id) && productEditing.value.field === field
}

function startProductEdit(product, field) {
  productEditing.value = { id: product?.id || '', field }
}

async function commitProductEdit(product) {
  productEditing.value = { id: '', field: '' }
  await updateProduct(product)
}

async function updateProductPrice(product, event) {
  const price = Math.max(0, Math.round(Number(event?.detail?.value || 0) * 100))
  product.price_cents = price
  productEditing.value = { id: '', field: '' }
  await updateProduct(product)
}

async function toggleProduct(product) {
  const wasPinned = isProductPinned(product)
  const nextStatus = product.status === 'on_sale' ? 'off_sale' : 'on_sale'
  product.status = nextStatus
  await updateProduct(product)
  if (nextStatus === 'off_sale' && wasPinned) {
    await merchantApi.unpinProduct(product.id).catch(() => {})
    await loadProducts({ reset: true })
  }
}

function hasPinnedSignal(product) {
  return Boolean(product?.pinned_at || product?.pinnedAt)
}

function isProductPinned(product) {
  return hasPinnedSignal(product)
}

function shouldShowPinControl(product) {
  if (isProductPinned(product)) return true
  if (product?.status !== 'on_sale') return false
  return merchantPinnedProductCount.value < maxPinnedProducts
}

async function toggleProductPinned(product) {
  if (!product?.id) return
  const wasPinned = isProductPinned(product)
  if (!wasPinned && merchantPinnedProductCount.value >= maxPinnedProducts) return
  merchantLoading.value = true
  try {
    if (wasPinned) {
      await merchantApi.unpinProduct(product.id)
      product.pinned_at = null
      product.pinnedAt = null
    } else {
      await merchantApi.pinProduct(product.id)
      product.pinned_at = new Date().toISOString()
    }
    await loadProducts({ reset: true })
    uni.showToast({ title: wasPinned ? '已取消置顶' : '已置顶', icon: 'none' })
  } catch (error) {
    uni.showToast({ title: error.message || '操作失败', icon: 'none' })
  } finally {
    merchantLoading.value = false
  }
}

async function deleteProduct(product) {
  const confirmed = await confirmAction(`确认删除「${product.name}」？`)
  if (!confirmed) return
  merchantLoading.value = true
  try {
    await merchantApi.deleteProduct(product.id)
    await loadProducts({ reset: true })
    uni.showToast({ title: '商品已删除', icon: 'none' })
  } catch (error) {
    uni.showToast({ title: error.message || '删除失败', icon: 'none' })
  } finally {
    merchantLoading.value = false
  }
}

async function chooseMerchantApplyPhoto() {
  await chooseMerchantImage((url) => { merchantApplyForm.photo_url = url }, 'stall_photo')
}

async function chooseJoinPhoto() {
  await chooseMerchantImage((url) => { joinForm.photo_url = url }, 'stall_photo')
}

async function chooseProductFormImage() {
  await chooseMerchantImage((url) => { productForm.image_url = url }, 'product')
}

async function chooseProductImage(product) {
  await chooseMerchantImage(async (url) => {
    product.image_url = url
    await updateProduct(product)
  }, 'product')
}

async function chooseSessionPhoto() {
  await chooseMerchantImage((url) => { sessionForm.photo_url = url }, 'stall_photo')
}

async function chooseMerchantImage(assign, preset = '') {
  merchantLoading.value = true
  try {
    const url = await chooseAndUploadImage('customer', { preset })
    await assign(url)
    uni.showToast({ title: '图片已上传', icon: 'success' })
  } catch (error) {
    if (!/cancel/i.test(error?.message || '')) uni.showToast({ title: error.message || '上传失败', icon: 'none' })
  } finally {
    merchantLoading.value = false
  }
}

function confirmAction(content) {
  return new Promise((resolve) => {
    uni.showModal({
      title: '请确认',
      content,
      confirmColor: '#2f1f0d',
      success: (res) => resolve(Boolean(res.confirm)),
      fail: () => resolve(false)
    })
  })
}

function openSessionSheet() {
  sessionForm.expected_end_at = localDateTime(new Date(Date.now() + 4 * 3600 * 1000))
  sessionForm.address = ''
  sessionForm.lat = 0
  sessionForm.lng = 0
  sessionForm.accuracy = 0
  sessionForm.photo_url = ''
  sessionLocationText.value = '正在自动定位，请保持页面打开…'
  sessionSheetOpen.value = true
  locateForSession()
}

async function locateForSession() {
  locating.value = true
  try {
    const location = await getUserLocation()
    sessionForm.lat = Number(location.lat)
    sessionForm.lng = Number(location.lng)
    sessionForm.accuracy = Number(location.accuracy || 0)
    clearAutoLocationAddress(sessionForm)
    sessionLocationText.value = `已自动定位，精度约 ${sessionForm.accuracy || '-'} 米，请补充位置描述。`
  } catch (error) {
    if (showDevLocation) {
      const location = devLocations[0]
      sessionForm.lat = location.lat
      sessionForm.lng = location.lng
      sessionForm.accuracy = location.accuracy
      sessionForm.address = location.name
      sessionLocationText.value = `${error.message || '定位失败'}，已使用开发定位：${location.name}`
    } else {
      sessionLocationText.value = error.message || '定位失败，请允许定位后重试。'
    }
  } finally {
    locating.value = false
  }
}

function clearAutoLocationAddress(form) {
  if (/^自动定位点/.test(String(form.address || '').trim())) form.address = ''
}

function cleanManualAddress(value) {
  const text = String(value || '').trim()
  return /^自动定位点/.test(text) ? '' : text
}

function chooseSessionEndTime(event) {
  const date = new Date()
  const [hour, minute] = String(event?.detail?.value || '').split(':').map(Number)
  date.setHours(hour || 0, minute || 0, 0, 0)
  if (date.getTime() <= Date.now()) date.setDate(date.getDate() + 1)
  sessionForm.expected_end_at = localDateTime(date)
}

async function startMerchantSession() {
  if (!hasSessionLocation.value) return uni.showToast({ title: '定位成功后可出摊', icon: 'none' })
  const address = cleanManualAddress(sessionForm.address)
  if (!address) return uni.showToast({ title: '请填写位置描述', icon: 'none' })
  merchantSessionLoading.value = true
  try {
    await merchantApi.startStallSession({
      lat: sessionForm.lat,
      lng: sessionForm.lng,
      address,
      expected_end_at: new Date(sessionForm.expected_end_at).toISOString(),
      location_accuracy: Math.round(sessionForm.accuracy || 0),
      photo_url: sessionForm.photo_url
    })
    sessionSheetOpen.value = false
    await loadMerchantHome()
    uni.showToast({ title: '已开始出摊', icon: 'success' })
  } catch (error) {
    uni.showToast({ title: error.message || '出摊失败', icon: 'none' })
  } finally {
    merchantSessionLoading.value = false
  }
}

async function endMerchantSession() {
  merchantSessionLoading.value = true
  try {
    await merchantApi.endStallSession()
    await loadMerchantHome()
    uni.showToast({ title: '已结束出摊', icon: 'none' })
  } catch (error) {
    uni.showToast({ title: error.message || '结束失败', icon: 'none' })
  } finally {
    merchantSessionLoading.value = false
  }
}

function shareTitle() {
  if (appMode.value === 'merchant') return `${merchantDisplayName.value}正在出摊`
  if (selectedVendor.value) return `${selectedVendor.value.name}正在出摊`
  return '附近流动摊位'
}

function localDateTime(date) {
  return new Date(date.getTime() - date.getTimezoneOffset() * 60000).toISOString().slice(0, 16)
}

function localTimeValue(date) {
  return localDateTime(date).slice(11, 16)
}

function timeText(value) {
  if (!value) return '--:--'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--'
  return `${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

async function submitJoin() {
  if (!joinForm.merchant_name || !joinForm.contact_phone || !joinForm.category) return uni.showToast({ title: '请填写摊位名、联系方式和类型', icon: 'none' })
  submittingForm.value = true
  try {
    await customerApi.createApplication({ ...joinForm })
    setJSONStorage('mplzCustomerContact', { name: joinForm.contact_name, phone: joinForm.contact_phone })
    await loadUserPageState({ silent: true }).catch(() => {})
    await loadMerchantStatus({ silent: true }).catch(() => {})
    appMode.value = userPageState.value?.page_mode === 'merchant' ? 'merchant' : 'application'
    joinCtaVisible.value = false
    uni.showToast({ title: '申请已提交', icon: 'success' })
    formSheet.value = ''
  } catch (error) {
    uni.showToast({ title: error.message || '提交失败', icon: 'none' })
  } finally { submittingForm.value = false }
}
async function submitFeedback() {
  if (!feedbackForm.description || !feedbackForm.contact_phone) return uni.showToast({ title: '请填写描述和联系方式', icon: 'none' })
  submittingForm.value = true
  try {
    await customerApi.createFeedback({ ...feedbackForm, source: 'customer', page_url: 'mp-weixin/pages/customer/index' })
    setJSONStorage('mplzCustomerContact', { name: feedbackForm.contact_name, phone: feedbackForm.contact_phone })
    uni.showToast({ title: '反馈已提交', icon: 'success' })
    formSheet.value = ''
  } catch (error) {
    uni.showToast({ title: error.message || '提交失败', icon: 'none' })
  } finally { submittingForm.value = false }
}
function shareQuery() {
  if (appMode.value === 'merchant' && merchantShareCode.value) return `shareCode=${encodeURIComponent(merchantShareCode.value)}`
  if (shareCode.value) return `shareCode=${encodeURIComponent(shareCode.value)}`
  if (selectedVendor.value?.merchantId) return `merchantId=${encodeURIComponent(selectedVendor.value.merchantId)}`
  return ''
}
function sharePath() {
  const query = shareQuery()
  return `/pages/customer/index${query ? `?${query}` : ''}`
}
</script>

<style scoped>
.customer-page {
  --customer-bg: #f5efe3;
  --customer-surface: #fffaf0;
  --customer-surface-2: #fff3d8;
  --customer-fg: #2f1f0d;
  --customer-muted: #7a6751;
  --customer-border: rgba(47, 31, 13, .14);
  --customer-accent: #f59e0b;
  min-height: 100vh;
  background:
    radial-gradient(circle at 8% 14%, rgba(245, 158, 11, .2), transparent 28%),
    linear-gradient(135deg, #f8f2e6, #efe1c7);
  color: var(--customer-fg);
  box-sizing: border-box;
  scrollbar-width: none;
}
button { padding: 0; margin: 0; line-height: normal; }
button::after { border: 0; }
.customer-page::-webkit-scrollbar,
.vendor-list-page::-webkit-scrollbar,
:global(html::-webkit-scrollbar),
:global(body::-webkit-scrollbar) {
  display: none;
}
:global(html),
:global(body) {
  scrollbar-width: none;
}
.merchant-flow-page,
.merchant-workbench-page {
  min-height: 100vh;
  padding: calc(var(--nav-title-top) + var(--nav-title-height) + 34rpx) 24rpx 132rpx;
  display: grid;
  gap: 16rpx;
  align-content: start;
  box-sizing: border-box;
}
.merchant-workbench-page {
  padding-top: calc(var(--nav-title-top) + var(--nav-title-height) + 126rpx);
  scrollbar-width: none;
}
.merchant-workbench-page::-webkit-scrollbar { display: none; }

.merchant-hero-card,
.merchant-panel {
  border: 2rpx solid var(--customer-border);
  border-radius: 38rpx;
  background: rgba(255,250,240,.86);
  box-shadow: 0 24rpx 70rpx rgba(47,31,13,.11);
  box-sizing: border-box;
}
.merchant-hero-card {
  min-height: 356rpx;
  padding: 32rpx;
  display: grid;
  gap: 24rpx;
  position: relative;
  overflow: hidden;
  background:
    radial-gradient(circle at 16% 16%, rgba(245,158,11,.34), transparent 30%),
    linear-gradient(135deg, rgba(255,250,240,.96), rgba(255,243,216,.84));
}
.merchant-hero-card.is-idle {
  place-items: center;
}
.merchant-hero-card.is-live {
  min-height: 420rpx;
  align-content: space-between;
  background:
    linear-gradient(180deg, rgba(47,31,13,.18), rgba(47,31,13,.66)),
    linear-gradient(135deg, #d9b981, #76512b);
  color: #fffaf0;
}
.merchant-hero-bg {
  position: absolute;
  inset: -28rpx;
  z-index: 0;
  width: calc(100% + 56rpx);
  height: calc(100% + 56rpx);
  filter: blur(4.5rpx) saturate(1.08);
  transform: scale(1.05);
  opacity: .72;
}
.merchant-hero-card.is-live::after {
  content: "";
  position: absolute;
  inset: 0;
  z-index: 1;
  background:
    radial-gradient(circle at 20% 16%, rgba(245,158,11,.28), transparent 34%),
    linear-gradient(180deg, rgba(47,31,13,.26), rgba(47,31,13,.74));
}
.merchant-hero-live-copy {
  position: relative;
  z-index: 2;
  justify-self: start;
  display: grid;
  gap: 12rpx;
}
.merchant-live-pill {
  width: fit-content;
  min-height: 48rpx;
  padding: 0 20rpx;
  border-radius: 999rpx;
  background: rgba(255,250,240,.18);
  color: #fffaf0;
  font-size: 24rpx;
  font-weight: 950;
  line-height: 48rpx;
  backdrop-filter: blur(12px);
}
.merchant-eyebrow {
  color: var(--customer-muted);
  font-size: 21rpx;
  font-weight: 950;
  text-transform: uppercase;
  letter-spacing: .08em;
}
.merchant-hero-card.is-live .merchant-eyebrow,
.merchant-hero-card.is-live .merchant-hero-copy { color: rgba(255,250,240,.86); }
.merchant-hero-title {
  color: inherit;
  font-size: 56rpx;
  line-height: 1;
  font-weight: 950;
}
.merchant-hero-copy,
.merchant-muted {
  color: var(--customer-muted);
  font-size: 25rpx;
  line-height: 1.45;
  font-weight: 800;
}
.merchant-status-row {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 14rpx;
  margin-top: 2rpx;
}
.merchant-status-card {
  gap: 18rpx;
  min-height: 176rpx;
  align-content: center;
  background:
    radial-gradient(circle at 8% 10%, rgba(245,158,11,.18), transparent 34%),
    linear-gradient(135deg, rgba(255,250,240,.96), rgba(255,247,232,.88));
}
.merchant-status-label {
  display: block;
  color: rgba(122,103,81,.78);
  font-size: 22rpx;
  font-weight: 950;
  text-align: left;
  letter-spacing: .08em;
}
.merchant-review-reason {
  display: grid;
  gap: 8rpx;
  padding: 18rpx;
  border-radius: 24rpx;
  background: rgba(245,158,11,.12);
  border: 2rpx solid rgba(245,158,11,.24);
}
.merchant-review-label {
  color: #92400e;
  font-size: 22rpx;
  font-weight: 950;
}
.merchant-status-title {
  display: block;
  margin: 10rpx 0 8rpx;
  color: var(--customer-fg);
  font-size: 38rpx;
  line-height: 1.12;
  font-weight: 950;
  letter-spacing: -.04em;
}
.merchant-status-pill {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 188rpx;
  min-height: 76rpx;
  padding: 0 34rpx;
  border-radius: 999rpx;
  background: rgba(245,158,11,.16);
  color: #92400e;
  font-size: 36rpx;
  font-weight: 950;
  letter-spacing: -.02em;
}
.merchant-status-pill.green { background: rgba(22,163,74,.13); color: #166534; }
.merchant-status-pill.red { background: rgba(220,38,38,.13); color: #991b1b; }
.merchant-panel {
  padding: 24rpx;
  display: grid;
  gap: 14rpx;
}
.merchant-section-head {
  display: grid;
  gap: 6rpx;
}
.merchant-section-title {
  display: block;
  color: var(--customer-fg);
  font-size: 31rpx;
  font-weight: 950;
}
.merchant-result-card.is-warning {
  border-color: rgba(245,158,11,.32);
  background: rgba(255,247,232,.9);
}
.application-info-card {
  background: rgba(255,250,240,.9);
}
.application-stall-photo {
  width: 100%;
  height: 280rpx;
  border-radius: 28rpx;
  background: var(--customer-surface-2);
}
.application-info-list {
  display: grid;
  overflow: hidden;
  border: 2rpx solid rgba(47,31,13,.07);
  border-radius: 26rpx;
  background: rgba(255,255,255,.42);
}
.application-info-row {
  display: grid;
  grid-template-columns: 168rpx minmax(0, 1fr);
  gap: 12rpx;
  padding: 18rpx 20rpx;
}
.application-info-row + .application-info-row {
  border-top: 2rpx solid rgba(47,31,13,.06);
}
.application-info-label {
  color: rgba(122,103,81,.78);
  font-size: 23rpx;
  font-weight: 900;
}
.application-info-value {
  color: var(--customer-fg);
  font-size: 25rpx;
  line-height: 1.36;
  font-weight: 900;
  word-break: break-word;
}
.next-step-card {
  background: linear-gradient(135deg, rgba(245,158,11,.12), rgba(255,250,240,.82));
}
.merchant-tabs {
  position: fixed;
  top: calc(var(--nav-title-top) + var(--nav-title-height) + 16rpx);
  left: 24rpx;
  right: 24rpx;
  z-index: 18;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10rpx;
  padding: 10rpx;
  border: 2rpx solid var(--customer-border);
  border-radius: 999rpx;
  background: rgba(255,250,240,.9);
  box-shadow: 0 18rpx 48rpx rgba(47,31,13,.1);
  backdrop-filter: blur(16px);
  box-sizing: border-box;
  transform: translateZ(0);
}
.merchant-tabs button {
  display: grid;
  place-items: center;
  height: 66rpx;
  line-height: 66rpx;
  border-radius: 999rpx;
  color: var(--customer-muted);
  font-size: 25rpx;
  font-weight: 950;
}
.merchant-tabs button.is-active {
  background: var(--customer-fg);
  color: #fffaf0;
}
.merchant-grid {
  display: grid;
  gap: 16rpx;
}
.merchant-hero-action,
.merchant-primary-btn {
  display: grid;
  place-items: center;
  min-width: 264rpx;
  height: 104rpx;
  line-height: 104rpx;
  border-radius: 999rpx;
  background: var(--customer-fg);
  color: #fffaf0;
  font-size: 31rpx;
  font-weight: 950;
  box-shadow: 0 18rpx 42rpx rgba(47,31,13,.2);
}
.merchant-hero-action {
  position: relative;
  z-index: 2;
  justify-self: center;
}
.merchant-hero-card.is-live .merchant-hero-action {
  background: #fffaf0;
  color: var(--customer-fg);
}
.merchant-mini-btn {
  display: inline-grid;
  place-items: center;
  min-height: 58rpx;
  padding: 0 20rpx;
  line-height: normal;
  border-radius: 999rpx;
  background: rgba(255,255,255,.58);
  color: var(--customer-fg);
  font-size: 23rpx;
  font-weight: 900;
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.08);
}
.merchant-mini-btn.is-on { background: rgba(22,163,74,.13); color: #166534; }
.merchant-mini-btn.is-pinned { background: rgba(245,158,11,.18); color: #92400e; }
.merchant-mini-btn.danger { background: rgba(220,38,38,.12); color: #991b1b; }
.merchant-pin-btn {
  display: inline-grid;
  place-items: center;
  width: 46rpx;
  height: 46rpx;
  min-width: 46rpx;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: #9a650a;
  box-shadow: none;
}
.merchant-pin-btn.is-pinned {
  background: transparent;
  color: #f5a50a;
  box-shadow: none;
}
.merchant-pin-btn.is-empty { opacity: .72; }
.merchant-pin-btn.is-floating {
  position: absolute;
  top: 20rpx;
  right: 22rpx;
  z-index: 4;
}
.merchant-pin-icon {
  width: 42rpx;
  height: 42rpx;
  display: block;
  filter: drop-shadow(0 4rpx 8rpx rgba(47,31,13,.16));
}
.merchant-add-fab {
  position: fixed;
  right: 24rpx;
  bottom: max(24rpx, env(safe-area-inset-bottom));
  z-index: 30;
  display: grid;
  place-items: center;
  width: 76rpx;
  height: 76rpx;
  padding: 0;
  border-radius: 999rpx;
  background: rgba(47,31,13,.92);
  color: #fffaf0;
  font-size: 46rpx;
  font-weight: 850;
  line-height: 76rpx;
  box-shadow: 0 18rpx 42rpx rgba(47,31,13,.2);
  backdrop-filter: blur(14px);
}
.merchant-product-form-sheet {
  padding-top: 18rpx;
}
.merchant-form-actions {
  display: grid;
  grid-template-columns: 160rpx minmax(0, 1fr);
  gap: 12rpx;
  align-items: center;
}
.merchant-form-actions .merchant-mini-btn,
.merchant-form-actions .submit-order {
  width: 100%;
  margin-top: 0;
}
.merchant-upload-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 112rpx;
  gap: 12rpx;
  align-items: center;
}
.merchant-image-preview {
  width: 100%;
  height: 220rpx;
  border-radius: 24rpx;
  background: var(--customer-surface-2);
}
.merchant-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20rpx;
}
.merchant-card-title-block {
  min-width: 0;
  display: grid;
  gap: 8rpx;
}
.merchant-qr {
  width: 340rpx;
  height: 340rpx;
  justify-self: center;
  border-radius: 24rpx;
  background: #fff;
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.08);
}
.merchant-qr-placeholder {
  display: grid;
  place-items: center;
  width: 340rpx;
  height: 340rpx;
  justify-self: center;
  border-radius: 24rpx;
  background: rgba(255,255,255,.56);
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.08);
  color: var(--customer-muted);
  font-size: 25rpx;
  font-weight: 900;
}
.merchant-actions-row {
  display: flex;
  justify-content: flex-end;
  padding: 0 8rpx;
}
.merchant-actions-row .merchant-primary-btn {
  min-width: 232rpx;
  height: 76rpx;
  font-size: 27rpx;
}
.merchant-product-list {
  display: grid;
  gap: 0;
  overflow: hidden;
  border: 2rpx solid var(--customer-border);
  border-radius: 44rpx;
  background: rgba(255,250,240,.84);
  box-shadow: 0 18rpx 52rpx rgba(47,31,13,.09);
  scrollbar-width: none;
}
.merchant-product-list::-webkit-scrollbar { display: none; }
.merchant-product-list-state {
  display: grid;
  place-items: center;
  min-height: 74rpx;
  margin-top: 0;
  color: var(--customer-muted);
  font-size: 23rpx;
  font-weight: 850;
  text-align: center;
}
.merchant-product-row {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: start;
  gap: 24rpx;
  padding: 24rpx;
}
.merchant-product-row + .merchant-product-row { border-top: 2rpx solid rgba(47,31,13,.08); }
.merchant-product-thumb {
  position: relative;
  display: grid;
  place-items: center;
  overflow: hidden;
  width: 100%;
  aspect-ratio: 4 / 3;
  height: auto;
  border-radius: 32rpx;
  background: var(--customer-surface-2);
  color: #9a650a;
  font-size: 42rpx;
  font-weight: 950;
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.08), 0 12rpx 28rpx rgba(47,31,13,.1);
}
.merchant-product-thumb image { width: 100%; height: 100%; }
.merchant-product-thumb-edit {
  position: absolute;
  left: 20rpx;
  right: 20rpx;
  bottom: 18rpx;
  min-height: 54rpx;
  padding: 0 16rpx;
  border-radius: 999rpx;
  background: rgba(47,31,13,.72);
  color: #fffaf0;
  font-size: 24rpx;
  font-weight: 900;
  line-height: 54rpx;
  text-align: center;
}
.merchant-product-main {
  min-width: 0;
  display: grid;
  min-height: 100%;
  grid-template-rows: auto auto 1fr;
  padding-right: 70rpx;
  gap: 18rpx;
}
.merchant-product-head { min-width: 0; }
.merchant-inline-input {
  width: 100%;
  min-height: 68rpx;
  padding: 0 16rpx;
  border-radius: 22rpx;
  background: rgba(255,255,255,.72);
  color: var(--customer-fg);
  font-size: 27rpx;
  font-weight: 900;
  box-sizing: border-box;
}
.merchant-inline-input.name { font-size: 31rpx; }
.merchant-inline-input.price { color: #b45309; }
.merchant-inline-value {
  display: block;
  width: 100%;
  min-height: 0;
  padding: 0;
  border-radius: 18rpx;
  background: transparent;
  color: var(--customer-fg);
  text-align: left;
  line-height: 1.18;
}
.merchant-product-name-button {
  font-size: 32rpx;
  font-weight: 950;
  overflow-wrap: anywhere;
}
.merchant-price-button {
  color: #b45309;
  font-size: 32rpx;
  font-weight: 950;
}
.merchant-product-price-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.merchant-product-actions {
  display: flex;
  align-items: center;
  align-self: end;
  gap: 10rpx;
  flex-wrap: wrap;
}
.identity-switch {
  position: fixed;
  left: 24rpx;
  bottom: max(30rpx, env(safe-area-inset-bottom));
  z-index: 30;
  display: grid;
  place-items: center;
  min-width: 126rpx;
  height: 56rpx;
  padding: 0 22rpx;
  border-radius: 999rpx;
  background: rgba(47,31,13,.88);
  color: #fffaf0;
  font-size: 22rpx;
  font-weight: 950;
  line-height: normal;
  box-shadow: 0 16rpx 38rpx rgba(47,31,13,.18);
  backdrop-filter: blur(14px);
}
.identity-switch.customer-side {
  bottom: max(24rpx, env(safe-area-inset-bottom));
  background: rgba(255,250,240,.92);
  color: var(--customer-fg);
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.1), 0 16rpx 38rpx rgba(47,31,13,.12);
}
.customer-page.is-map-tab { height: 100vh; overflow: hidden; }
.c-panel {
  border: 2rpx solid var(--customer-border);
  border-radius: 48rpx;
  background: rgba(255, 250, 240, .84);
  box-shadow: 0 28rpx 76rpx rgba(47, 31, 13, .09);
}
.c-btn {
  min-height: 84rpx;
  padding: 0 32rpx;
  border-radius: 999rpx;
  border: 0;
  font-size: 28rpx;
  font-weight: 850;
}
.c-btn.primary { background: var(--customer-fg); color: #fffaf0; }
.c-btn.secondary { border: 2rpx solid var(--customer-border); background: rgba(255,255,255,.55); color: var(--customer-fg); }
.customer-toolbar {
  position: fixed;
  top: var(--toolbar-top);
  left: 22rpx;
  right: 22rpx;
  z-index: 18;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 66rpx 66rpx;
  align-items: start;
  gap: 12rpx;
  padding: 12rpx;
  border: 2rpx solid var(--customer-border);
  border-radius: 36rpx;
  background: rgba(255, 250, 240, .9);
  box-shadow: 0 24rpx 64rpx rgba(47,31,13,.1);
  backdrop-filter: blur(16px);
  box-sizing: border-box;
}
.customer-page-title {
  position: fixed;
  top: var(--nav-title-top);
  left: 36rpx;
  right: var(--nav-title-right);
  z-index: 19;
  height: var(--nav-title-height);
  color: var(--customer-fg);
  font-size: 36rpx;
  font-weight: 950;
  line-height: var(--nav-title-height);
  letter-spacing: -.04em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.toolbar-search { position: relative; min-width: 0; }
.c-input {
  width: 100%;
  height: 66rpx;
  min-height: 66rpx;
  padding: 0 66rpx 0 24rpx;
  border: 2rpx solid var(--customer-border);
  border-radius: 26rpx;
  background: rgba(255,255,255,.56);
  color: var(--customer-fg);
  font-size: 26rpx;
  box-sizing: border-box;
}
.search-clear {
  position: absolute;
  right: 10rpx;
  top: 9rpx;
  display: grid;
  place-items: center;
  width: 48rpx;
  height: 48rpx;
  padding: 0;
  border-radius: 999rpx;
  background: rgba(47,31,13,.08);
  color: var(--customer-muted);
  font-size: 32rpx;
  line-height: 48rpx;
}
.view-toggle,
.favorite-entry {
  display: grid;
  place-items: center;
  width: 66rpx;
  height: 66rpx;
  padding: 0;
  border: 0;
  border-radius: 24rpx;
  background: var(--customer-fg);
  color: #fffaf0;
  font-size: 30rpx;
  font-weight: 950;
  box-shadow: 0 16rpx 36rpx rgba(47,31,13,.14);
}
.favorite-entry { position: relative; overflow: hidden; background: rgba(255,255,255,.6); color: var(--customer-fg); box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.1), 0 16rpx 36rpx rgba(47,31,13,.07); transform-origin: center; }
.favorite-entry.is-opening { animation: favorite-entry-open .48s cubic-bezier(.18,.86,.18,1) both; }
.favorite-entry.is-opening .toolbar-star-icon { animation: favorite-entry-star .48s cubic-bezier(.18,.86,.18,1) both; }
.view-toggle image,
.modal-map-action image {
  width: 40rpx;
  height: 40rpx;
}
.category-strip { grid-column: 1 / -1; width: 100%; white-space: nowrap; }
.category-strip ::-webkit-scrollbar { display: none; }
.category-track { display: flex; gap: 12rpx; padding: 2rpx 2rpx 0; }
.category-pill {
  flex: 0 0 auto;
  display: inline-flex;
  align-items: center;
  gap: 10rpx;
  min-height: 58rpx;
  padding: 6rpx 18rpx 6rpx 8rpx;
  border: 2rpx solid rgba(47,31,13,.1);
  border-radius: 999rpx;
  background: rgba(255,255,255,.54);
  color: var(--customer-fg);
  font-size: 22rpx;
  font-weight: 900;
}
.category-pill.is-active { border-color: rgba(245,158,11,.62); background: rgba(245,158,11,.2); }
.category-icon {
  display: inline-grid;
  place-items: center;
  width: 44rpx;
  height: 44rpx;
  border-radius: 17rpx;
  color: #fffaf0;
  font-size: 20rpx;
  font-weight: 950;
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.12);
}
.category-icon image { width: 32rpx; height: 32rpx; }
.dev-location-select {
  grid-column: 1 / -1;
  min-height: 58rpx;
  padding: 0 20rpx;
  border: 2rpx dashed rgba(47,31,13,.28);
  border-radius: 28rpx;
  background: rgba(255,255,255,.7);
  color: var(--customer-fg);
  font-size: 23rpx;
  font-weight: 900;
  line-height: 58rpx;
  box-sizing: border-box;
}
.vendor-list-page {
  width: 100%;
  min-height: 100vh;
  padding: calc(340rpx + env(safe-area-inset-top)) 24rpx 132rpx;
  display: grid;
  align-content: start;
  gap: 14rpx;
  box-sizing: border-box;
  scrollbar-width: none;
}
.vendor-card-list { display: grid; grid-auto-rows: max-content; align-content: start; align-items: start; gap: 12rpx; }
.vendor-card {
  position: relative;
  display: grid;
  align-self: start;
  grid-template-columns: 188rpx minmax(0, 1fr);
  gap: 14rpx;
  padding: 14rpx;
  border-radius: 30rpx;
  box-sizing: border-box;
}
.vendor-rank-photo {
  display: grid;
  place-items: center;
  overflow: hidden;
  width: 188rpx;
  height: 152rpx;
  min-height: 152rpx;
  border-radius: 24rpx;
  background: var(--customer-surface-2);
  color: #9a650a;
  font-size: 46rpx;
  font-weight: 950;
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.08);
}
.vendor-rank-photo image { width: 100%; height: 100%; }
.vendor-rank-photo.is-placeholder { background: rgba(245,158,11,.1); }
.vendor-card-main { min-width: 0; display: grid; gap: 8rpx; align-content: start; }
.vendor-title { display: block; padding-right: 38rpx; font-size: 28rpx; line-height: 1.12; font-weight: 950; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.favorite-star {
  position: absolute;
  top: 15rpx;
  right: 17rpx;
  z-index: 3;
  display: grid;
  place-items: center;
  width: 46rpx;
  height: 46rpx;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: #9a650a;
  line-height: 1;
}
.favorite-star.is-active { color: #f5a50a; }
.favorite-star.is-tapping .favorite-star-icon,
.modal-favorite-star.is-tapping .favorite-star-icon {
  animation: favorite-star-pop .42s cubic-bezier(.2,.85,.2,1) both;
}
.favorite-star-icon, .toolbar-star-icon {
  display: block;
  width: 42rpx;
  height: 42rpx;
  filter: drop-shadow(0 4rpx 8rpx rgba(47,31,13,.16));
}
.toolbar-star-icon {
  width: 38rpx;
  height: 38rpx;
}
.vendor-subtitle { display: flex; flex-wrap: wrap; gap: 8rpx; color: var(--customer-muted); font-size: 21rpx; font-weight: 850; }
.vendor-subtitle text + text::before { content: "·"; margin-right: 8rpx; }
.vendor-address { color: var(--customer-muted); font-size: 21rpx; font-weight: 850; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.vendor-products { display: flex; align-items: center; flex-wrap: wrap; gap: 8rpx; }
.vendor-products text { min-height: 38rpx; padding: 0 13rpx; border-radius: 999rpx; background: rgba(245,158,11,.12); color: var(--customer-fg); font-size: 19rpx; font-weight: 900; line-height: 38rpx; }
.empty-panel { display: grid; place-items: center; align-self: start; gap: 16rpx; min-height: 414rpx; padding: 40rpx; color: var(--customer-muted); font-size: 28rpx; text-align: center; box-sizing: border-box; }
.empty-title { display: block; color: var(--customer-fg); font-size: 34rpx; font-weight: 950; }
.empty-copy { display: block; margin-top: 10rpx; }
.location-notice-card {
  align-self: start;
  display: grid;
  gap: 18rpx;
  margin-bottom: 4rpx;
  padding: 26rpx;
  border-radius: 34rpx;
  background: linear-gradient(135deg, rgba(255,250,240,.9), rgba(255,243,216,.9));
  box-sizing: border-box;
}
.location-notice-actions {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14rpx;
}
.location-notice-actions .c-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 72rpx;
  min-height: 72rpx;
  padding: 0 18rpx;
  border-radius: 999rpx;
  font-size: 25rpx;
  font-weight: 950;
  line-height: 1;
  box-sizing: border-box;
}
.location-notice-actions .c-btn.secondary {
  background: rgba(255,250,240,.78);
  box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.1);
}
.map-screen { position: relative; width: 100%; height: 100vh; overflow: hidden; }
.native-map { position: absolute; inset: 0; z-index: 0; width: 100%; height: 100%; }
.map-status {
  position: fixed;
  left: 36rpx;
  bottom: max(36rpx, env(safe-area-inset-bottom));
  z-index: 11;
  max-width: 390rpx;
  padding: 20rpx 28rpx;
  border: 2rpx solid var(--customer-border);
  border-radius: 999rpx;
  background: rgba(255,250,240,.9);
  color: var(--customer-muted);
  font-size: 24rpx;
  font-weight: 850;
}
.detail-close { position: absolute; top: 24rpx; right: 24rpx; z-index: 2; display: grid; place-items: center; width: 58rpx; height: 58rpx; padding: 0; border-radius: 999rpx; background: rgba(255,250,240,.84); color: var(--customer-fg); font-size: 38rpx; font-weight: 900; line-height: 58rpx; box-shadow: 0 12rpx 30rpx rgba(47,31,13,.12); }
.detail-photo { position: relative; overflow: hidden; height: 360rpx; border-radius: 36rpx; background: var(--customer-surface-2); }
.detail-photo image { width: 100%; height: 100%; }
.detail-photo::after { content: ""; position: absolute; inset: 0; background: linear-gradient(180deg, rgba(47,31,13,.05) 0%, rgba(47,31,13,.28) 46%, rgba(47,31,13,.78) 100%); }
.detail-placeholder { display: grid; place-items: center; height: 100%; color: #9a650a; font-size: 72rpx; font-weight: 950; }
.detail-photo-info { position: absolute; left: 28rpx; right: 28rpx; bottom: 28rpx; z-index: 1; display: grid; gap: 8rpx; color: #fffaf0; font-size: 26rpx; font-weight: 850; text-shadow: 0 4rpx 28rpx rgba(47,31,13,.55); }
.detail-title { font-size: 44rpx; line-height: 1.12; font-weight: 950; }
.detail-actions { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16rpx; }
.sheet-mask { position: fixed; inset: 0; z-index: 50; display: flex; align-items: flex-end; background: rgba(47,31,13,.32); animation: customer-mask-fade .18s ease-out both; }
.bottom-sheet { width: 100%; max-height: 86vh; padding: 18rpx 24rpx calc(28rpx + env(safe-area-inset-bottom)); border-radius: 38rpx 38rpx 0 0; border: 2rpx solid var(--customer-border); background: var(--customer-surface); box-shadow: 0 -28rpx 80rpx rgba(47,31,13,.2); box-sizing: border-box; overflow-y: auto; }
.sheet-handle { width: 86rpx; height: 9rpx; margin: 0 auto 18rpx; border-radius: 999rpx; background: rgba(47,31,13,.2); }
.sheet-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 18rpx; }
.sheet-title { display: block; color: var(--customer-fg); font-size: 36rpx; font-weight: 950; }
.sheet-subtitle, .product-desc { display: block; color: var(--customer-muted); font-size: 24rpx; line-height: 1.45; }
.close-button, .sheet-actions-row button { height: 66rpx; line-height: 66rpx; border-radius: 999rpx; background: rgba(255,255,255,.62); color: var(--customer-fg); font-size: 24rpx; font-weight: 850; }
.close-button { width: 110rpx; }
.sheet-actions-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12rpx; margin-top: 18rpx; }
.state-card { margin-top: 20rpx; padding: 28rpx; border-radius: 30rpx; background: rgba(255,255,255,.46); color: var(--customer-muted); font-size: 27rpx; }
.stall-products-sheet { position: relative; display: flex; flex-direction: column; height: 78vh; max-height: 1120rpx; padding: 18rpx 18rpx calc(28rpx + env(safe-area-inset-bottom)); overflow: hidden; overscroll-behavior: contain; animation: product-sheet-enter .28s cubic-bezier(.18,.86,.18,1) both; transform-origin: center bottom; }
.modal-map-action { position: absolute; top: 24rpx; left: 24rpx; z-index: 2; display: grid; place-items: center; width: 58rpx; height: 58rpx; padding: 0; border-radius: 999rpx; background: rgba(255,250,240,.84); color: var(--customer-fg); box-shadow: 0 12rpx 30rpx rgba(47,31,13,.12); }
.modal-nav-action { background: rgba(255,250,240,.92); }
.stall-products-hero { position: relative; overflow: hidden; flex: 0 0 360rpx; height: 360rpx; border-radius: 34rpx; background: var(--customer-surface-2); }
.stall-products-hero image { width: 100%; height: 100%; }
.stall-products-hero.is-placeholder { display: grid; place-items: center; background: rgba(245,158,11,.12); }
.stall-products-hero.is-placeholder > image { width: 144rpx; height: 144rpx; }
.stall-products-hero::after { content: ""; position: absolute; inset: 0; background: linear-gradient(180deg, rgba(47,31,13,.02) 0%, rgba(47,31,13,.3) 50%, rgba(47,31,13,.82) 100%); }
.stall-products-sheet .modal-map-action,
.stall-products-sheet .detail-close { animation: product-action-enter .24s ease-out .15s both; }
.stall-products-sheet .stall-products-hero { animation: product-hero-enter .34s cubic-bezier(.18,.86,.18,1) .04s both; transform-origin: center bottom; }
.stall-products-hero-info { position: absolute; left: 28rpx; right: 28rpx; bottom: 24rpx; z-index: 1; color: #fffaf0; text-shadow: 0 4rpx 24rpx rgba(47,31,13,.55); animation: product-copy-enter .3s ease-out .13s both; }
.stall-products-title { display: block; font-size: 38rpx; line-height: 1.12; font-weight: 950; }
.stall-products-meta-row { display: grid; grid-template-columns: minmax(0, 1fr) 58rpx; align-items: center; gap: 14rpx; margin-top: 10rpx; font-size: 24rpx; font-weight: 850; }
.stall-products-meta-copy { min-width: 0; display: grid; gap: 4rpx; }
.stall-products-meta-copy text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.stall-products-end { font-size: 22rpx; color: rgba(255,250,240,.88); }
.modal-favorite-star { display: grid; place-items: center; width: 46rpx; height: 46rpx; padding: 0; border: 0; border-radius: 0; background: transparent; color: #9a650a; box-shadow: none; }
.modal-favorite-star.is-active { color: #f5a50a; }
.stall-products-list-scroll { flex: 1; min-height: 0; height: 100%; margin-top: 16rpx; overflow: hidden; overscroll-behavior: contain; box-sizing: border-box; }
.stall-products-list-scroll::-webkit-scrollbar { display: none; width: 0; height: 0; }
.stall-products-list { display: grid; gap: 12rpx; min-height: 318rpx; padding-bottom: calc(96rpx + env(safe-area-inset-bottom)); box-sizing: border-box; scrollbar-width: none; }
.stall-products-list::-webkit-scrollbar { display: none; width: 0; height: 0; }
.stall-products-list .stall-product-row { animation: product-row-enter .28s cubic-bezier(.2,.8,.2,1) both; }
.stall-products-list .stall-product-row:nth-child(1) { animation-delay: 90ms; }
.stall-products-list .stall-product-row:nth-child(2) { animation-delay: 115ms; }
.stall-products-list .stall-product-row:nth-child(3) { animation-delay: 140ms; }
.stall-products-list .stall-product-row:nth-child(4) { animation-delay: 165ms; }
.stall-products-list .stall-product-row:nth-child(5) { animation-delay: 190ms; }
.stall-products-list .stall-product-row:nth-child(6) { animation-delay: 215ms; }
.stall-products-list .stall-product-row:nth-child(n+7) { animation-delay: 230ms; }
.stall-product-row { display: flex; align-items: center; gap: 14rpx; padding: 13rpx; border-radius: 24rpx; background: rgba(255,255,255,.52); }
.stall-product-thumb { display: grid; place-items: center; overflow: hidden; flex: 0 0 106rpx; width: 106rpx; height: 98rpx; border-radius: 20rpx; background: rgba(245,158,11,.18); }
.stall-product-thumb.can-preview { transition: transform .16s ease, filter .16s ease; }
.stall-product-thumb.can-preview:active { transform: scale(.96); filter: brightness(.96); }
.stall-product-thumb image { width: 100%; height: 100%; }
.stall-product-thumb.is-placeholder image { width: 58rpx; height: 58rpx; }
.stall-product-content { flex: 1; min-width: 0; }
.product-name { display: block; color: var(--customer-fg); font-size: 28rpx; font-weight: 950; }
.product-price { display: block; margin-top: 6rpx; color: #b45309; font-size: 28rpx; font-weight: 950; }
.product-preview-mask {
  position: fixed;
  inset: 0;
  z-index: 80;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 36rpx;
  background: rgba(47,31,13,.42);
  backdrop-filter: blur(14rpx);
  -webkit-backdrop-filter: blur(14rpx);
  box-sizing: border-box;
  animation: product-preview-mask-in .18s ease-out both;
}
.product-preview-dialog {
  position: relative;
  width: calc(100vw - 72rpx);
  max-width: 680rpx;
  padding: 0;
  border: 0;
  border-radius: 32rpx;
  background: transparent;
  box-shadow: 0 34rpx 90rpx rgba(47,31,13,.34);
  box-sizing: border-box;
  overflow: hidden;
  animation: product-preview-dialog-in .26s cubic-bezier(.18,.86,.18,1) both;
}
.product-preview-close {
  position: absolute;
  top: 20rpx;
  right: 20rpx;
  z-index: 2;
  display: grid;
  place-items: center;
  width: 62rpx;
  height: 62rpx;
  padding: 0;
  border-radius: 999rpx;
  background: rgba(47,31,13,.68);
  color: #fffaf0;
  font-size: 38rpx;
  font-weight: 950;
  line-height: 62rpx;
  box-shadow: 0 12rpx 30rpx rgba(47,31,13,.22);
}
.product-preview-image {
  display: block;
  width: 100%;
  height: auto;
  border-radius: 32rpx;
  background: transparent;
}
.product-preview-title {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 1;
  display: block;
  padding: 60rpx 28rpx 24rpx;
  color: #fffaf0;
  font-size: 28rpx;
  font-weight: 950;
  line-height: 1.35;
  text-align: center;
  text-shadow: 0 4rpx 22rpx rgba(47,31,13,.56);
  background: linear-gradient(180deg, rgba(47,31,13,0), rgba(47,31,13,.62));
  box-sizing: border-box;
}
.stall-products-skeleton .skeleton-product-row { background: rgba(255,255,255,.38); animation-name: product-row-enter, product-skeleton-pulse; animation-duration: .28s, 1.2s; animation-timing-function: cubic-bezier(.2,.8,.2,1), ease-in-out; animation-iteration-count: 1, infinite; animation-fill-mode: both, none; }
.skeleton-thumb { background: rgba(245,158,11,.16); }
.skeleton-line { height: 18rpx; border-radius: 999rpx; background: rgba(122,103,81,.14); }
.skeleton-line.is-title { width: 54%; height: 24rpx; }
.skeleton-line.is-desc { width: 82%; margin-top: 12rpx; }
.skeleton-line.is-price { width: 30%; height: 22rpx; margin-top: 14rpx; background: rgba(180,83,9,.16); }
.simple-form { display: grid; gap: 14rpx; margin-top: 18rpx; }
.simple-form input, .simple-form textarea, .form-picker { width: 100%; min-height: 76rpx; padding: 0 22rpx; border-radius: 22rpx; background: #fffaf0; color: var(--customer-fg); font-size: 27rpx; box-sizing: border-box; }
.simple-form textarea { min-height: 160rpx; padding-top: 18rpx; }
.form-picker { line-height: 76rpx; font-weight: 850; }
.form-picker.is-placeholder { color: rgba(122,103,81,.72); font-weight: 700; }
.submit-order { width: 100%; height: 82rpx; line-height: 82rpx; margin-top: 18rpx; border-radius: 999rpx; background: var(--customer-accent); color: var(--customer-fg); font-size: 28rpx; font-weight: 950; }
.favorites-sheet { position: relative; height: 86vh; max-height: 1240rpx; padding-top: 108rpx; overflow: hidden; }
.favorites-search { position: absolute; left: 24rpx; right: 110rpx; top: 24rpx; }
.favorites-list-scroll { height: 100%; }
.favorite-card-list { display: grid; gap: 12rpx; padding-bottom: 12rpx; }
.favorite-swipe-row {
  position: relative;
  overflow: hidden;
  border-radius: 28rpx;
  animation: favorite-card-enter .28s cubic-bezier(.2,.8,.2,1) both;
}
.favorite-card-list .favorite-swipe-row:nth-child(1) { animation-delay: 20ms; }
.favorite-card-list .favorite-swipe-row:nth-child(2) { animation-delay: 45ms; }
.favorite-card-list .favorite-swipe-row:nth-child(3) { animation-delay: 70ms; }
.favorite-card-list .favorite-swipe-row:nth-child(4) { animation-delay: 95ms; }
.favorite-card-list .favorite-swipe-row:nth-child(5) { animation-delay: 120ms; }
.favorite-card-list .favorite-swipe-row:nth-child(6) { animation-delay: 145ms; }
.favorite-card-list .favorite-swipe-row:nth-child(7) { animation-delay: 170ms; }
.favorite-card-list .favorite-swipe-row:nth-child(8) { animation-delay: 195ms; }
.favorite-card-list .favorite-swipe-row:nth-child(n+9) { animation-delay: 210ms; }
.favorite-remove-action {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 1;
  display: grid;
  place-items: center;
  width: 140rpx;
  height: 100%;
  padding: 0;
  border: 0;
  border-radius: 0 28rpx 28rpx 0;
  background: linear-gradient(135deg, #B45309, #92400E);
  color: #fffaf0;
  font-size: 25rpx;
  font-weight: 950;
  line-height: 1;
  opacity: 0;
  pointer-events: none;
  transform: translateX(18rpx);
  transition: opacity .16s ease, transform .16s ease;
}
.favorite-swipe-row.is-open .favorite-remove-action { opacity: 1; pointer-events: auto; transform: translateX(0); }
.favorite-card {
  position: relative;
  z-index: 2;
  display: grid;
  grid-template-columns: 112rpx minmax(0, 1fr);
  gap: 14rpx;
  padding: 12rpx;
  border-radius: 28rpx;
  background: #fffaf0;
  border: 2rpx solid rgba(47,31,13,.08);
  box-sizing: border-box;
  transition: transform .18s ease;
}
.favorite-swipe-row.can-swipe .favorite-card { filter: saturate(.88); }
.favorite-swipe-row.is-open .favorite-card { box-shadow: 0 16rpx 36rpx rgba(47,31,13,.12); }
.favorites-join-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 52rpx;
  gap: 12rpx;
  align-items: center;
  margin-top: 4rpx;
  padding: 16rpx 14rpx 16rpx 18rpx;
  border-radius: 28rpx;
  border: 2rpx dashed rgba(245,158,11,.38);
  background: linear-gradient(135deg, rgba(245,158,11,.16), rgba(255,255,255,.58));
  box-sizing: border-box;
}
.favorites-join-copy { min-width: 0; display: grid; gap: 8rpx; }
.favorites-join-title { display: block; color: var(--customer-fg); font-size: 27rpx; font-weight: 950; }
.favorites-join-muted { display: block; color: var(--customer-muted); font-size: 21rpx; line-height: 1.35; font-weight: 850; }
.favorites-join-close {
  display: grid;
  place-items: center;
  width: 52rpx;
  height: 52rpx;
  padding: 0;
  border-radius: 999rpx;
  background: rgba(255,250,240,.72);
  color: rgba(47,31,13,.5);
  font-size: 30rpx;
  font-weight: 900;
  line-height: 52rpx;
}
.favorite-thumb { display: grid; place-items: center; overflow: hidden; width: 112rpx; height: 100rpx; border-radius: 22rpx; background: rgba(245,158,11,.12); }
.favorite-thumb image { width: 100%; height: 100%; }
.favorite-thumb.is-placeholder image { width: 58rpx; height: 58rpx; }
.favorite-card-main { min-width: 0; display: grid; align-items: center; }
.card-head { display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: 12rpx; align-items: start; }
.favorite-title { display: block; color: var(--customer-fg); font-size: 27rpx; font-weight: 950; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.favorite-muted { display: block; margin-top: 8rpx; color: var(--customer-muted); font-size: 21rpx; font-weight: 850; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.status-pill { display: inline-flex; align-items: center; min-height: 38rpx; padding: 0 14rpx; border-radius: 999rpx; background: rgba(47,31,13,.08); color: var(--customer-muted); font-size: 20rpx; font-weight: 900; white-space: nowrap; }
.status-pill.open { background: rgba(22,163,74,.13); color: #166534; }
@keyframes customer-mask-fade {
  from { opacity: 0; }
  to { opacity: 1; }
}
@keyframes customer-sheet-rise {
  from { opacity: 0; transform: translateY(72rpx) scale(.985); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes product-sheet-enter {
  0% { opacity: 0; transform: translateY(86rpx) scale(.975); }
  72% { opacity: 1; transform: translateY(-4rpx) scale(1.002); }
  100% { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes product-hero-enter {
  from { opacity: 0; transform: translateY(18rpx) scale(.985); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes product-copy-enter {
  from { opacity: 0; transform: translateY(16rpx); }
  to { opacity: 1; transform: translateY(0); }
}
@keyframes product-action-enter {
  from { opacity: 0; transform: translateY(10rpx) scale(.92); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes product-row-enter {
  from { opacity: 0; transform: translateY(18rpx) scale(.985); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes product-skeleton-pulse {
  0%, 100% { opacity: .68; }
  50% { opacity: 1; }
}
@keyframes product-preview-mask-in {
  from { opacity: 0; }
  to { opacity: 1; }
}
@keyframes product-preview-dialog-in {
  from { opacity: 0; transform: translateY(26rpx) scale(.96); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@keyframes favorite-star-pop {
  0% { opacity: .74; transform: scale(.72) rotate(-10deg); }
  45% { opacity: 1; transform: scale(1.26) rotate(8deg); }
  72% { transform: scale(.92) rotate(-3deg); }
  100% { opacity: 1; transform: scale(1) rotate(0); }
}
@keyframes favorite-entry-open {
  0% { transform: translateY(0) scale(1); box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.1), 0 16rpx 36rpx rgba(47,31,13,.07); }
  38% { transform: translateY(-4rpx) scale(1.12); box-shadow: inset 0 0 0 2rpx rgba(245,158,11,.28), 0 22rpx 46rpx rgba(180,83,9,.18); }
  72% { transform: translateY(1rpx) scale(.96); }
  100% { transform: translateY(0) scale(1); box-shadow: inset 0 0 0 2rpx rgba(47,31,13,.1), 0 16rpx 36rpx rgba(47,31,13,.07); }
}
@keyframes favorite-entry-star {
  0% { transform: scale(1) rotate(0); }
  42% { transform: scale(1.24) rotate(-12deg); }
  72% { transform: scale(.9) rotate(8deg); }
  100% { transform: scale(1) rotate(0); }
}
@keyframes favorite-card-enter {
  from { opacity: 0; transform: translateY(20rpx) scale(.985); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}
@media (max-width: 360px) {
  .vendor-card { grid-template-columns: 170rpx minmax(0, 1fr); gap: 12rpx; padding: 12rpx; }
  .vendor-rank-photo { width: 170rpx; height: 140rpx; min-height: 140rpx; }
  .vendor-list-title { font-size: 50rpx; }
}
</style>
