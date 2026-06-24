# 后端接口与业务逻辑说明（给前端使用）

## 1. 后端目录职责

| 目录 | 职责 | 前端关注点 |
| --- | --- | --- |
| `backend/cmd/server` | 程序入口：加载配置、注册模型、可选 seed、启动 Fiber。 | 部署和启动方式。 |
| `backend/internal/api` | 业务路由注册。 | 接口契约入口。 |
| `backend/internal/model` | 数据模型、JSON 投影、简单写入 hook。 | 请求/响应字段、自动联动规则。 |
| `backend/internal/query` | 列表/详情查询参数和 DB 过滤。 | GET query 参数、公开数据边界。 |
| `backend/internal/service` | 复杂业务：出摊、附近聚合、审核、反馈、订单状态机、分享图。 | 非 CRUD 接口背后的流程。 |
| `backend/internal/conf` | `custom` 配置映射。 | 分享域名、前端代理目录、商品/收藏数量限制。 |
| `backend/internal/bootstrap` | 页面代理、schema 准备、demo seed。 | 本地/合并部署、演示数据。 |
| `backend/gkk` | 本地 gkk 框架：鉴权、通用 handler、上传、微信、后台 URM/RBAC、错误和分页。 | 通用响应、token、上传、微信登录和后台接口规则。 |
| `backend/internal/contracttest` | 静态契约测试。 | 后端维护用，不是接口。 |

忽略 `backend/gkk/.gocache`、`backend/server` 等构建/缓存产物。

## 2. 统一约定

### 2.1 路由分组

`backend/internal/api/register.go` 注册：

```text
POST /api/upload                         前台登录
/api/pub/*                               公开接口
/api/customer/*                          前台登录 + user_id scope
/api/merchant/*                          前台登录 + 商户身份；/me 后的接口带 merchant_id scope
/api/sys/*                               gkk 后台管理与业务后台
/api/Wx/*                                gkk 微信/小程序能力
/                                         custom.frontend 配置后代理前端 dist
```

### 2.2 鉴权头

所有受保护接口统一使用：

```http
token: <jwt>
```

不要使用 `Authorization: Bearer ...`。

### 2.3 响应规则

- 列表：`{ "data": [], "total": 0 }`
- 写接口成功：HTTP 200，body 为空
- 前端封装必须允许成功响应为空 body，不能强制 JSON parse
- 错误响应来自 gkk `expect`，常见结构：

```json
{
  "code": 10003,
  "message": "参数校验失败",
  "ui": { "action": "toast", "target": "upload" },
  "data": {}
}
```

`ui`、`data` 不是每个错误都有。

### 2.4 分页参数

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `page` | number | 默认 1 |
| `size` | number | 默认 20，最大 300 |
| `start` | string/date | 使用 `Period` 的接口支持 `created_at >= start` |
| `end` | string/date | 使用 `Period` 的接口支持 `created_at < end + 1 day` |

## 3. 后端配置对前端的影响

`backend/config.yaml` 的 `custom` 映射到 `backend/internal/conf/conf.go`。

```yaml
custom:
  share_url: "http://localhost:5173"
  seed_demo_data: true
  frontend: "../frontend/dist"
  max_merchant_products: 50
  max_customer_favorites: 50
```

| 配置 | 作用 | 默认 |
| --- | --- | --- |
| `share_url` | 拼接商户 H5 分享链接：`{share_url}/share/{share_code}` | `http://localhost:5173` |
| `frontend` | Go 服务代理前端 dist 到 `/`；空值则不代理页面 | 不代理 |
| `seed_demo_data` | 启动时填充 demo 数据 | false |
| `max_merchant_products` | 单个商户可创建商品总数 | 50 |
| `max_customer_favorites` | 单个顾客可收藏商户总数 | 50 |

数量限制超出时返回业务校验错误。更新已有商品、重复收藏命中已有记录，不增加数量。

小程序登录和小程序码使用 gkk 顶层 `auth.mini`：

```yaml
auth:
  mini:
    appid: "小程序 appid"
    key: "小程序 secret"
```

小程序码页面固定为 `pages/customer/index`，`env_version` 固定为 `release`。

## 4. 登录和用户页面模式

### 4.1 小程序登录

仅当 `auth.mini.appid` 和 `auth.mini.key` 都配置时注册。

```http
POST /api/Wx/mini
Content-Type: application/json
```

请求：

```json
{
  "code": "wx.login 返回的 code",
  "phone": "可选",
  "nickname": "可选",
  "avatar": "可选"
}
```

响应通常为：

```json
{
  "token": "jwt",
  "expire": "2026-06-23 18:30",
  "valid": true,
  "phone": false
}
```

内部逻辑：

1. 用 code 调微信 `code2session`。
2. 通过 `open_id` 查找用户。
3. 不存在则创建 `model.User`，默认 `page_mode=customer`。
4. 写入 `open_id`。
5. 用户不可用时拒绝登录。
6. 返回 type=`mini` 的 JWT。

### 4.2 公众号/扫码相关微信接口

gkk 微信模块接口：

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/Wx` | 微信服务器校验 |
| `POST` | `/api/Wx` | 微信消息回调 |
| `POST` | `/api/Wx/qr` | 生成扫码登录状态；要求线上微信配置 |
| `GET` | `/api/Wx/qr?key=` | 查询扫码状态 |
| `GET` | `/api/Wx/login` | 公众号 OAuth 跳转 |
| `GET` | `/api/Wx/openid` | OpenID 跳转 |
| `GET` | `/api/Wx/ticket?key=` | 扫码 ticket 换 token |
| `GET` | `/api/Wx/share?url=` | JS-SDK 分享配置 |
| `POST/GET/DELETE` | `/api/Wx/img` | 微信素材上传/列表/删除 |

小程序端正常只需要 `/api/Wx/mini`。

### 4.3 登录前台用户

```http
GET /api/customer/me
token: <jwt>
```

响应是登录用户 `model.User`：

```json
{
  "id": 21,
  "username": "...",
  "nickname": "...",
  "avatar": "...",
  "phone": "...",
  "merchant_id": 21,
  "page_mode": "merchant"
}
```

前端按 `page_mode` 分流：

| `page_mode` | 页面模式 | 后续调用 |
| --- | --- | --- |
| `customer` | 顾客首页 | 调公开附近/收藏等接口 |
| `application` | 申请中页面 | 调 `/api/customer/applications` |
| `merchant` | 商户端 | 调 `/api/merchant/me`、商品、出摊接口 |

## 5. 上传

```http
POST /api/upload
token: <jwt>
Content-Type: multipart/form-data
```

表单：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `file` | 是 | 图片/音频/视频/常见办公文档，受 gkk allowlist 限制 |
| `preset` | 否 | 图片业务尺寸预设；不传则保持原文件上传 |

`preset` 支持：

| preset | 输出尺寸 | 适用场景 |
| --- | --- | --- |
| `stall_photo` | `1280x720` | 商户开始出摊时的营业照片 |
| `product` | `800x600` | 商品列表/商品详情图片 |

传入以上 `preset` 时，后端会对 `jpg/jpeg/png/gif` 先等比缩放到覆盖目标尺寸，再中心裁切，并以 `.jpg` 上传到 OSS。

响应：

```json
"https://oss.example.com/user/nearby/uuid.jpg"
```

内部逻辑：上传到 OSS 的 `user/nearby` 目录。OSS 配置不完整会导致上传失败。

## 6. 公开接口 `/api/pub`

### 6.1 附近营业摊位

```http
GET /api/pub/stalls/nearby
```

Query：

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `lat` | number | 否 | 用户纬度；和 `lng` 同时存在才计算距离 |
| `lng` | number | 否 | 用户经度 |
| `limit` | number | 否 | 默认 50，最大 300 |
| `page` / `size` | number | 否 | gkk 分页；通常用 `limit` 即可 |
| `q` | string | 否 | 搜索商户名、分类、公告、出摊地址 |
| `category` | string | 否 | 单分类，也支持逗号分隔 |
| `categories` | string[] | 否 | 多分类，可重复传参 |
| `min_lat/max_lat/min_lng/max_lng` | number | 否 | 四个都有才按视口过滤 |
| `zoom` | string | 否 | 预留参数，响应不回显 |

响应：

```json
{
  "data": [
    {
      "merchant": {
        "id": 21,
        "display_name": "gkk的维修铺",
        "category": "其他摊位",
        "avatar_url": "https://...png",
        "announcement": "",
        "products": [
          { "name": "芒果班戟", "price_cents": 1800 }
        ]
      },
      "stall_session": {
        "status": "active",
        "lat": 36.65184,
        "lng": 117.12009,
        "address": "自动定位点",
        "photo_url": "https://...jpg",
        "location_accuracy": 65,
        "started_at": "2026-06-22T16:15:33+08:00",
        "expected_end_at": "2026-06-22T20:15:00+08:00"
      },
      "distance_meters": 620,
      "walk_minutes": 8,
      "display_status": "active",
      "entry_mode": "nearby",
      "location_accuracy": 65,
      "last_online_at": "2026-06-22T16:15:33+08:00"
    }
  ],
  "total": 1
}
```

内部逻辑：

- 只查 `stall_sessions.status=active`。
- 只返回 `merchant.status=active` 且 `merchant.verify_status=verified` 的商户。
- 支持视口、分类、搜索过滤。
- 只返回公开商户投影，不暴露手机号、用户 id、审核字段等。
- 有用户定位时，DB 分页后再按距离排序。
- 卡片商品来自 `merchants.products`，不是实时查询全量商品。

### 6.2 公开商户详情

```http
GET /api/pub/merchants/detail?id=21
GET /api/pub/merchants/detail?share_code=XUOKTK
```

Query：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 与 `share_code` 二选一 | 商户主键 |
| `share_code` | 与 `id` 二选一 | 商户固定分享码/小程序 scene |

响应：

```json
{
  "id": 21,
  "display_name": "gkk的维修铺",
  "category": "其他摊位",
  "avatar_url": "https://...png",
  "announcement": "",
  "products": [
    { "name": "芒果班戟", "price_cents": 1800 }
  ]
}
```

内部逻辑：只解析 `active + verified` 商户。

### 6.3 单商户出摊状态

```http
GET /api/pub/merchants/stall?id=21
```

Query：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 是 | 商户 id |

响应：找到则返回公开 `StallSession` 投影；未找到时 `OptionalFirst` 返回 200 空 body。

```json
{
  "status": "active",
  "lat": 36.65184,
  "lng": 117.12009,
  "address": "自动定位点",
  "photo_url": "https://...jpg",
  "location_accuracy": 65,
  "started_at": "2026-06-22T16:15:33+08:00",
  "expected_end_at": "2026-06-22T20:15:00+08:00"
}
```

内部逻辑：必须是未过期 active 出摊，且商户 active/verified。

### 6.4 单商户公开商品列表

```http
GET /api/pub/merchants/products?id=21
GET /api/pub/merchants/products?merchant_id=21&name=咖啡&page=1&size=20
```

Query：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| `id` | 否 | 推荐传商户 id |
| `merchant_id` | 否 | 商户 id；`id` 为空时使用 |
| `name` | 否 | 商品名/描述搜索 |
| `page/size` | 否 | 分页 |

响应：

```json
{
  "data": [
    {
      "id": 1,
      "name": "冰美式",
      "description": "清爽低糖",
      "price_cents": 1800,
      "stock": 9999,
      "image_url": "https://...jpg"
    }
  ],
  "total": 1
}
```

内部逻辑：只返回 active/verified 商户的 `on_sale` 商品；置顶商品优先。

## 7. 顾客接口 `/api/customer`

全部需要 `token`，路由组自动注入 `user_id`，`user_id` 由后端注入。

### 7.1 登录用户

```http
GET /api/customer/me
```

见 4.3。

### 7.2 登录用户申请详情

```http
GET /api/customer/applications?id=3
```

`id` 可选；不传 id 时返回登录用户 scope 下第一条申请。

响应示例：

```json
{
  "id": 3,
  "created_at": "2026-06-18T15:19:45+08:00",
  "merchant_name": "gkk的维修铺",
  "contact_name": "gkk",
  "contact_phone": "13165418517",
  "category": "其他摊位",
  "photo_url": "https://...png",
  "usual_area": "省博物馆周边3km",
  "remark": "家电维修等",
  "status": "pending",
  "review_reason": ""
}
```

### 7.3 创建/更新入驻申请

```http
PUT /api/customer/applications
Content-Type: application/json
```

Body：

```json
{
  "id": 3,
  "merchant_name": "gkk的维修铺",
  "contact_name": "gkk",
  "contact_phone": "13165418517",
  "category": "其他摊位",
  "photo_url": "https://oss...png",
  "usual_area": "省博物馆周边3km",
  "remark": "家电维修, 手机维修"
}
```

规则：

- `id` 为空或 0：创建。
- `id` 存在：更新登录用户 scope 下申请。
- `user_id/status/created_at` 服务端控制。
- 创建时用户 `page_mode` 改成 `application`。
- 成功返回 200 空 body。

### 7.4 提交反馈

```http
POST /api/customer/feedback
```

Body：

```json
{
  "source": "customer",
  "contact_name": "张三",
  "contact_phone": "13800138000",
  "description": "页面打不开",
  "image_url": "https://...png",
  "page_url": "/nearby"
}
```

规则：

- `source` 必须是 `customer` 或 `merchant`。
- `contact_phone`、`description` 必填。
- `status` 固定由后端置为 `pending`。
- `source=customer` 时清空 `merchant_id`。
- 成功返回 200 空 body。

### 7.5 收藏列表

```http
GET /api/customer/favorites?name=咖啡&page=1&size=20
```

Query：

| 参数 | 说明 |
| --- | --- |
| `name` | 按商户名称 `display_name like %name%` 过滤收藏项。 |
| `page/size` | 分页 |

响应：

```json
{
  "data": [
    {
      "id": 9,
      "created_at": "2026-06-20T10:00:00+08:00",
      "merchant_id": 21,
      "merchant": {
        "id": 21,
        "display_name": "gkk的维修铺",
        "category": "其他摊位",
        "avatar_url": "https://...png",
        "announcement": "",
        "products": [
          { "name": "芒果班戟", "price_cents": 1800 }
        ]
      },
      "stall_session": {
        "status": "active",
        "lat": 36.65184,
        "lng": 117.12009,
        "address": "自动定位点",
        "photo_url": "",
        "location_accuracy": 65,
        "started_at": "2026-06-22T16:15:33+08:00",
        "expected_end_at": "2026-06-22T20:15:00+08:00"
      }
    }
  ],
  "total": 1
}
```

内部逻辑：只查登录用户收藏；`name` 参与主查询过滤；商户返回公开字段投影，并预加载该商户未过期 active 出摊。

### 7.6 添加收藏

```http
POST /api/customer/favorites
```

Body：

```json
{ "merchant_id": 21 }
```

规则：

- `user_id` 后端注入。
- `FirstOrCreate`，重复收藏幂等成功。
- 新建收藏时受 `custom.max_customer_favorites` 限制，默认 50。
- 成功返回 200 空 body。

### 7.7 删除收藏

```http
DELETE /api/customer/favorites?id=9
```

`id` 是收藏记录 id，不是商户 id。成功返回 200 空 body。

## 8. 商户接口 `/api/merchant`

全部需要前台 token 且登录用户必须有 `merchant_id`。

### 8.1 商户资料

```http
GET /api/merchant/me
```

响应：

```json
{
  "created_at": "2026-06-18T15:19:45+08:00",
  "updated_at": "2026-06-19T18:11:45+08:00",
  "id": 21,
  "display_name": "gkk的维修铺",
  "phone": "13165418517",
  "category": "其他摊位",
  "avatar_url": "https://...png",
  "announcement": "",
  "contact_phone": "13165418517",
  "products": [
    { "name": "芒果班戟", "price_cents": 1800 }
  ],
  "share_code": "XUOKTK",
  "share_url": "http://localhost:5173/share/XUOKTK",
  "share_poster_url": "https://oss...png",
  "share_qrcode_url": "https://oss...png",
  "share_qrcode_channel": "mini_program",
  "status": "active",
  "verify_status": "verified",
  "disabled_reason": ""
}
```

内部逻辑：

- 按登录用户 `merchant_id` 查询。
- `share_url` 由后端拼接，不落库。
- `share_qrcode_url` 与 `share_poster_url` 值一致。
- `/me` 返回已保存的分享图地址。

### 8.2 更新商户资料

```http
PUT /api/merchant/me
```

Body：

```json
{
  "id": 21,
  "display_name": "gkk的维修铺",
  "category": "其他摊位",
  "avatar_url": "https://...png",
  "announcement": "今日在省博附近",
  "contact_phone": "13165418517"
}
```

服务端保护字段：`user_id`、`share_code`、`share_poster_url`、`share_qrcode_channel`、`phone`、`products`、`status`、`verify_status`、`created_at`。成功返回 200 空 body。

### 8.3 出摊记录

```http
GET /api/merchant/stalls?id=1
GET /api/merchant/stalls?status=active&page=1&size=20
```

Query 支持：`id`、`status`、`page/size/start/end`。该接口使用 gkk `Get`：有 `id` 时查详情，无 `id` 时查列表；都带登录商户 scope。

### 8.4 开始出摊

```http
POST /api/merchant/stalls/start
```

Body：

```json
{
  "lat": 36.65184,
  "lng": 117.12009,
  "address": "自动定位点 36.65184, 117.12009",
  "photo_url": "https://...jpg",
  "location_accuracy": 65,
  "expected_end_at": "2026-06-22T20:15:00+08:00"
}
```

规则：

- `lat/lng/address` 必填。
- `expected_end_at` 不传则默认当前时间 + 6 小时。
- `expected_end_at` 必须晚于当前服务端时间。
- 新出摊开始前，会关闭登录商户已有 active 出摊。
- 成功返回 200 空 body。

### 8.5 结束出摊

```http
POST /api/merchant/stalls/end
```

不需要 body。

内部逻辑：

1. 找登录商户最新 active 出摊。
2. 设置 `status=ended`，写入 `ended_at`。
3. 将该商户所有 `pending_accept` 订单关闭为 `expired`。
4. 回补这些订单占用的库存。
5. 成功返回 200 空 body。

### 8.6 商品列表

```http
GET /api/merchant/products?name=咖啡&status=on_sale&page=1&size=20
```

Query：

| 参数 | 说明 |
| --- | --- |
| `id` | 商品 id，来自嵌入的 `Period[uint]` |
| `name` | 名称 LIKE 搜索 |
| `status` | `on_sale` / `off_sale` |
| `min_price_cents` | 最低价格 |
| `max_price_cents` | 最高价格 |
| `stock_less` | 库存小于该值 |
| `page/size/start/end` | 分页/时间过滤 |

响应：

```json
{
  "data": [
    {
      "id": 1,
      "created_at": "2026-06-20T10:00:00+08:00",
      "updated_at": "2026-06-20T10:00:00+08:00",
      "name": "冰美式",
      "description": "清爽低糖",
      "price_cents": 1800,
      "stock": 9999,
      "image_url": "https://...jpg",
      "status": "on_sale",
      "pinned_at": "2026-06-22T10:00:00+08:00",
      "sort_order": 3
    }
  ],
  "total": 1
}
```

内部逻辑：登录商户 scope；排序为置顶优先、置顶时间倒序、sort_order 倒序、id 倒序。

### 8.7 创建/更新商品

```http
PUT /api/merchant/products
```

创建 body（不传 `id` 或 `id=0`）：

```json
{
  "name": "冰美式",
  "description": "清爽低糖",
  "price_cents": 1800,
  "stock": 9999,
  "image_url": "https://...jpg",
  "status": "on_sale",
  "sort_order": 3
}
```

更新 body（传 `id`）：

```json
{
  "id": 1,
  "name": "冰美式",
  "description": "清爽低糖",
  "price_cents": 1800,
  "stock": 9999,
  "image_url": "https://...jpg",
  "status": "off_sale",
  "sort_order": 3
}
```

规则：

- `merchant_id`、`pinned_at`、`created_at` 服务端控制。
- `name`、`image_url` 必填。
- `status` 默认 `on_sale`，合法值：`on_sale`、`off_sale`。
- `price_cents`、`stock` 不能为负数。
- 新建商品受 `custom.max_merchant_products` 限制，默认 50。
- 创建/更新/删除都会刷新 `merchant.products` 摘要。
- 下架商品自动取消置顶。
- 成功返回 200 空 body。

### 8.8 删除商品

```http
DELETE /api/merchant/products?id=1
```

`id` 为商品主键。删除后刷新 `merchant.products`。成功返回 200 空 body。

### 8.9 置顶商品

```http
PUT /api/merchant/products/pin
```

Body：

```json
{ "id": 1 }
```

规则：

- 只有 `on_sale` 商品可置顶。
- `pinned_at` 由服务端生成。
- 单商户最多 3 个置顶商品。
- 置顶第 4 个时，最早置顶的商品会自动取消置顶。
- 置顶后刷新 `merchant.products`，最新置顶在前。
- 成功返回 200 空 body。

### 8.10 取消置顶商品

```http
PUT /api/merchant/products/unpin
```

Body：

```json
{ "id": 1 }
```

规则：清空 `pinned_at`，刷新 `merchant.products`；置顶不足 3 个时从上架商品补足摘要。成功返回 200 空 body。

## 9. 后台接口 `/api/sys`

### 9.1 gkk 后台登录/资料

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `GET` | `/api/sys/sms` | 无 | 发送短信验证码 |
| `POST` | `/api/sys/sms` | 无 | 手机号登录 |
| `POST` | `/api/sys/code` | 无 | 验证码检查 |
| `POST` | `/api/sys/pwd` | 无 | 账号密码登录 |
| `POST` | `/api/sys/renew` | 后台 | 续期 token |
| `GET` | `/api/sys/profile` | 后台 | 登录后台用户 |
| `PUT` | `/api/sys/user/info` | 后台 | 更新后台用户资料 |
| `PUT` | `/api/sys/user/password` | 后台 | 修改后台密码 |
| `GET` | `/api/sys/user/ifAdmin` | 后台 | 是否超级管理员 |

账号密码登录 body：

```json
{
  "account": "admin",
  "password": "admin123",
  "code": "123456"
}
```

### 9.2 gkk URM/RBAC 管理接口

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| `GET` | `/api/sys/user/list` | 用户列表 |
| `GET` | `/api/sys/user/all` | 用户简表 |
| `GET` | `/api/sys/user?id=1` | 用户详情 |
| `POST` | `/api/sys/user` | 新增用户 |
| `PUT` | `/api/sys/user` | 编辑用户 |
| `PUT` | `/api/sys/user/status` | 更新用户状态 |
| `DELETE` | `/api/sys/user?id=1` | 删除用户 |
| `GET` | `/api/sys/menu/param` | 菜单/角色参数 |
| `GET` | `/api/sys/menu/all` | 菜单简表 |
| `GET` | `/api/sys/menu/list` | 菜单列表 |
| `GET` | `/api/sys/menu?id=1` | 菜单详情 |
| `POST` | `/api/sys/menu` | 新增菜单 |
| `PUT` | `/api/sys/menu` | 编辑菜单 |
| `DELETE` | `/api/sys/menu?id=1` | 删除菜单 |
| `GET` | `/api/sys/role/all` | 角色简表 |
| `GET` | `/api/sys/role/list` | 角色列表 |
| `GET` | `/api/sys/role/rule` | 路由权限树 |
| `GET` | `/api/sys/role?id=1` | 角色详情 |
| `POST` | `/api/sys/role` | 新增角色 |
| `PUT` | `/api/sys/role` | 编辑角色 |
| `DELETE` | `/api/sys/role?id=1` | 删除角色 |
| `GET` | `/api/sys/role/user` | 角色用户 |
| `DELETE` | `/api/sys/role/user` | 移除角色用户 |

### 9.3 业务后台接口

全部需要后台 token + RBAC 权限。

#### 入驻申请列表

```http
GET /api/sys/applications?status=pending&merchant_name=维修&page=1&size=20
```

支持 query：`application_no`、`user_id`、`contact_phone`、`merchant_name`、`category`、`status`、`merchant_id`、`page/size/start/end`。

#### 入驻申请详情

```http
GET /api/sys/applications/detail?id=3
```

返回 `Application`，存在商户时预加载 `Merchant`。

#### 审核入驻申请

```http
POST /api/sys/applications/:id/approve
POST /api/sys/applications/:id/reject
```

Body：

```json
{ "review_reason": "资料完整" }
```

通过逻辑：锁定申请 -> 确认未通过 -> 创建/确认商户 -> 生成/确认分享图 -> 设置申请通过和审核人 -> 绑定用户 `merchant_id` -> 用户 `page_mode=merchant`。

拒绝逻辑：锁定申请 -> 已通过则不允许改 -> 设置拒绝状态和审核人 -> 用户保持 `page_mode=application`。

#### 商户列表

```http
GET /api/sys/merchants?display_name=咖啡&category=咖啡饮品&status=active&verify_status=verified&page=1&size=20
```

支持 query：`user_id`、`phone`、`display_name`、`category`、`status`、`verify_status`、`page/size/start/end`。

#### 更新商户状态

```http
PUT /api/sys/merchants/status
```

Body：

```json
{
  "id": 21,
  "status": "disabled",
  "disabled_reason": "违规"
}
```

`status` 只能是 `active` 或 `disabled`；恢复 active 会清空 `disabled_reason`。

#### 订单列表

```http
GET /api/sys/orders?status=pending_accept&merchant_id=21&category=咖啡饮品&page=1&size=20
```

后台订单接口支持 query：`order_no`、`merchant_id`、`stall_session_id`、`user_id`、`customer_phone`、`status`、`payment_status`、`category`、`page/size/start/end`。返回预加载 `Merchant`、`StallSession`、`Items` 的订单列表。

#### 后台取消/退款订单

```http
POST /api/sys/orders/:id/cancel
POST /api/sys/orders/:id/refund
```

无 body。取消会校验状态机并回补库存；退款只在 `paid/refunding` 状态下标记为 `refunded`，已退款重复调用视为成功。

#### 反馈列表

```http
GET /api/sys/feedback?source=customer&status=pending&page=1&size=20
```

支持 query：`source`、`user_id`、`merchant_id`、`contact_phone`、`description`、`status`、`handler_id`、`page/size/start/end`。

#### 处理反馈

```http
PUT /api/sys/feedback/:id
```

Body：

```json
{
  "status": "resolved",
  "handler_note": "已处理"
}
```

`status` 只能是 `pending`、`handling`、`resolved`、`closed`。状态为 `resolved/closed` 时写入 `handled_at`，否则清空。

#### 活跃出摊列表

```http
GET /api/sys/stalls/active?merchant_id=21&category=咖啡饮品&q=省博&page=1&size=20
```

只返回未过期 active 出摊，且商户必须 active/verified。支持 `merchant_id`、`category`、`q`、分页和时间过滤。

## 10. 前端必须理解的内部逻辑

### 10.1 `merchant.products` 商品摘要

公开商户卡片不实时查全量商品，而是读取 `merchants.products` JSON 摘要。

后端自动维护：

- 创建商品：刷新摘要。
- 更新商品：刷新摘要。
- 删除商品：刷新摘要。
- 置顶商品：最多保留 3 个置顶，刷新摘要。
- 取消置顶：刷新摘要，置顶不足 3 个时从上架商品补足。
- 商品下架：自动取消置顶并刷新摘要。

前端卡片展示使用 `merchant.products`；需要完整商品图文时再调用 `/api/pub/merchants/products`。

### 10.2 分享码和分享图

- `share_code` 直接存在 `merchants` 表，稳定不变。
- `share_url` 后端按 `custom.share_url` 拼接，不落库。
- `share_poster_url` 存在商户表，`share_qrcode_url` 输出值等于 `share_poster_url`。
- 审核通过流程会生成/确认分享图。
- `/api/merchant/me` 返回已保存的分享图地址。
- 小程序扫码场景：scene 使用 `share_code`，进入页面后调用 `/api/pub/merchants/detail?share_code=...`。

### 10.3 出摊生命周期

- 商户正常只有一个 active 出摊：开始新出摊前会关闭已有 active 出摊。
- 后台 cron 每分钟执行一次，`expected_end_at <= now` 的 active 出摊会自动改为 ended。
- 手动结束出摊会同时关闭未接单订单并回补库存。

### 10.4 公开数据边界

公开接口默认只返回前端展示必需字段：

- 公开商户：`id/display_name/category/avatar_url/announcement/products`
- 公开出摊：状态、经纬度、地址、照片、精度、开始/预计结束时间
- 公开商品：商品展示字段

手机号、用户 id、审核字段、owner 字段不会出现在公开接口里。

## 11. 接口总表

| 方法 | 路径 | 鉴权 | 用途 |
| --- | --- | --- | --- |
| `POST` | `/api/Wx/mini` | 无，配置完整才有 | 小程序登录 |
| `GET` | `/api/Wx/login` | 无 | 公众号 OAuth |
| `GET` | `/api/Wx/share` | 无 | JS-SDK 分享配置 |
| `POST` | `/api/upload` | 前台登录 | 上传文件 |
| `GET` | `/api/pub/stalls/nearby` | 无 | 附近 active 出摊 |
| `GET` | `/api/pub/merchants/detail` | 无 | 公开商户详情 |
| `GET` | `/api/pub/merchants/stall` | 无 | 单商户 active 出摊 |
| `GET` | `/api/pub/merchants/products` | 无 | 单商户公开商品 |
| `GET` | `/api/customer/me` | 前台登录 | 登录用户和 page_mode |
| `GET` | `/api/customer/applications` | 前台登录 | 登录用户申请详情 |
| `PUT` | `/api/customer/applications` | 前台登录 | 创建/更新入驻申请 |
| `POST` | `/api/customer/feedback` | 前台登录 | 提交反馈 |
| `GET` | `/api/customer/favorites` | 前台登录 | 收藏列表 |
| `POST` | `/api/customer/favorites` | 前台登录 | 添加收藏 |
| `DELETE` | `/api/customer/favorites` | 前台登录 | 删除收藏 |
| `GET` | `/api/merchant/me` | 商户 | 商户资料和分享信息 |
| `PUT` | `/api/merchant/me` | 商户 | 更新商户资料 |
| `GET` | `/api/merchant/stalls` | 商户 | 出摊记录/详情 |
| `POST` | `/api/merchant/stalls/start` | 商户 | 开始出摊 |
| `POST` | `/api/merchant/stalls/end` | 商户 | 结束出摊，无 body |
| `GET` | `/api/merchant/products` | 商户 | 商品列表 |
| `PUT` | `/api/merchant/products` | 商户 | 创建/更新商品 |
| `DELETE` | `/api/merchant/products` | 商户 | 删除商品 |
| `PUT` | `/api/merchant/products/pin` | 商户 | 置顶商品 |
| `PUT` | `/api/merchant/products/unpin` | 商户 | 取消置顶 |
| `GET` | `/api/sys/applications` | 后台/RBAC | 申请列表 |
| `GET` | `/api/sys/applications/detail` | 后台/RBAC | 申请详情 |
| `POST` | `/api/sys/applications/:id/approve` | 后台/RBAC | 审核通过 |
| `POST` | `/api/sys/applications/:id/reject` | 后台/RBAC | 审核拒绝 |
| `GET` | `/api/sys/merchants` | 后台/RBAC | 商户列表 |
| `PUT` | `/api/sys/merchants/status` | 后台/RBAC | 更新商户状态 |
| `GET` | `/api/sys/orders` | 后台/RBAC | 订单列表 |
| `POST` | `/api/sys/orders/:id/cancel` | 后台/RBAC | 后台取消订单 |
| `POST` | `/api/sys/orders/:id/refund` | 后台/RBAC | 后台标记退款 |
| `GET` | `/api/sys/feedback` | 后台/RBAC | 反馈列表 |
| `PUT` | `/api/sys/feedback/:id` | 后台/RBAC | 处理反馈 |
| `GET` | `/api/sys/stalls/active` | 后台/RBAC | 活跃出摊列表 |

## 12. 集成注意事项

- `GET /api/merchant/stalls` 是 gkk `Get`：有 id 走详情，无 id 走列表。
- `POST /api/merchant/stalls/end` 不绑定 JSON，不需要 body。
- 所有写接口成功都可能是空 body，前端请求封装必须把 HTTP 200 当成功。
