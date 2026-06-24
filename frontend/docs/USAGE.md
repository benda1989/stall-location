# frontend 使用文档

本文档说明当前 `frontend` uni-app 项目的安装、开发、构建、真机调试和常见问题处理。

## 环境要求

- Node.js：建议 18+
- npm：随 Node 安装即可
- 微信开发者工具：用于导入 `mp-weixin` 构建产物

## 安装依赖

```bash
cd frontend
npm install
```

## 环境变量

环境变量文件：

```text
.env.example
.env.development
.env.production
.env.local
.env.*.local
```

核心变量：

```bash
VITE_API_BASE=
```

本地调试可临时使用顾客 token：

```bash
VITE_CUSTOMER_TOKEN=调试用token
```

建议写入 `.env.development.local`，该文件已被 `.gitignore` 忽略。

## 常用命令

### H5 本地调试

```bash
cd frontend
npm run dev:h5
```

默认启动 uni H5 开发服务，可用浏览器访问终端输出的本地地址。

### 微信小程序开发模式

```bash
cd frontend
npm run dev:mp-weixin
```

产物目录：

```text
frontend/dist/dev/mp-weixin
```

在微信开发者工具中导入该目录。

### 微信小程序生产构建

```bash
cd frontend
npm run build:mp-weixin
```

产物目录：

```text
frontend/dist/build/mp-weixin
```

在微信开发者工具中导入该目录，用于预览、上传和真机验证。

### H5 构建

```bash
cd frontend
npm run build:h5
```

产物目录：

```text
frontend/dist/build/h5
```

## 微信开发者工具导入说明

不要直接导入 `frontend` 源码目录。应根据运行命令导入对应产物：

- 执行 `npm run dev:mp-weixin` 后导入 `frontend/dist/dev/mp-weixin`
- 执行 `npm run build:mp-weixin` 后导入 `frontend/dist/build/mp-weixin`

如果看到类似错误：

```text
ENOENT: no such file or directory, open '.../dist/dev/mp-weixin/project.config.json'
```

通常是导入目录和执行命令不匹配。比如执行了 `build:mp-weixin`，但开发者工具仍导入 `dist/dev/mp-weixin`。

## 登录与 token

小程序正常流程：

1. 调用 `uni.login({ provider: 'weixin' })` 获取 code。
2. 请求 `POST /api/Wx/mini` 换取顾客 token。
3. 后续顾客、商户接口在 header 中携带 `token`。
4. 如果接口返回 401 且提示登录，会清理 token 并重新登录。

本地 H5 或开发联调可使用：

```bash
VITE_CUSTOMER_TOKEN=xxx
```

配置后会优先使用该 token，便于绕过微信登录调试页面。

## 定位与导航

顾客端列表打开时会先尝试定位：

- 定位成功：带 `lat/lng` 请求附近商户。
- 定位失败：页面显示提示卡片，提供“重新定位”和“去设置”，同时降级加载默认商户列表。

商户端开始出摊：

- 必须定位成功后才显示“开始出摊”按钮。
- 系统不会自动填入 `自动定位点 ...`，需要商户手动填写位置描述。
- 出摊照片会作为商户工作台出摊状态卡片的虚化背景。

真机定位失败时优先检查：

- 手机系统定位是否开启。
- 微信 App 是否有定位权限。
- 小程序右上角设置中是否允许位置信息。
- 小程序后台隐私保护指引是否已配置并发布。

## 图片上传

统一使用：

```text
POST /api/upload
```

代码入口：

```text
src/api/client.js
chooseAndUploadImage()
uploadFile()
```

用于：

- 入驻申请摊位图片
- 商户出摊照片
- 商品图片

## 主要业务入口

### 顾客端

文件：

```text
src/pages/customer/index.vue
src/api/customer.js
src/utils/location.js
src/utils/catalog.js
```

常见链路：

- 附近商户：`customerApi.nearbyStalls()`
- 商户详情商品：`customerApi.listProducts(merchantId)`
- 收藏列表：`customerApi.listFavorites({ name })`
- 收藏/取消：`customerApi.addFavorite()` / `customerApi.removeFavorite()`
- 入驻申请：`customerApi.createApplication()`
- 反馈：`customerApi.createFeedback()`

### 商户端

文件：

```text
src/pages/customer/index.vue
src/api/merchant.js
```

常见链路：

- 商户状态：`merchantApi.getApplicationStatus()`
- 商户总览：`merchantApi.getDashboard()`
- 商品列表：`merchantApi.listProducts()`
- 商品置顶：`merchantApi.pinProduct()` / `merchantApi.unpinProduct()`
- 开始/结束出摊：`merchantApi.startStallSession()` / `merchantApi.endStallSession()`

## 发布前检查清单

- `npm run build:mp-weixin` 可以通过。
- 微信开发者工具导入的是 `frontend/dist/build/mp-weixin`。
- AppID 为 ``。
- request 合法域名包含 ``。
- 隐私保护指引包含定位用途。
- 真机验证登录、定位、收藏、商品详情、导航、入驻申请、商户出摊、商品置顶。

## 常见问题

### 页面白屏

优先检查最近是否加入了小程序不兼容的全局 WXSS 选择器。避免在小程序全局样式中使用过强的：

```css
*::-webkit-scrollbar
view::-webkit-scrollbar
```

当前项目只在安全容器上隐藏滚动条。

### 请求没有带位置

顾客列表首次加载会先定位。如果仍未带 `lat/lng`，检查：

- 是否拒绝了定位权限。
- 是否打开的是开发定位模式。
- `userLocation` 是否已被设置。
- 微信开发者工具是否模拟了定位。

### 接口 401

401 通常代表 token 不可用。当前请求层会在需要登录时重新走小程序登录。若本地使用 `VITE_CUSTOMER_TOKEN`，请确认 token 未过期且角色兼容。

### 开发者工具提示 project.config.json 不存在

确认导入目录和命令匹配：

- `dev:mp-weixin` -> `dist/dev/mp-weixin`
- `build:mp-weixin` -> `dist/build/mp-weixin`
