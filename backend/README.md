# Stall Location Backend

`backend` 是「出摊啦」的 Go 服务，负责前台登录、公开摊位查询、顾客收藏/申请/反馈、商户资料/出摊/商品管理、后台业务管理、上传、微信/小程序能力和可选前端静态代理。

## 技术栈

- Go + Fiber v3
- GORM + PostgreSQL
- 本地 `gkk` 框架：JWT、RBAC、通用 CRUD handler、错误响应、OSS 上传、微信/小程序登录
- 阿里云 OSS 上传
- uni-app 前端通过 `/api` 调用后端

## 目录职责

| 目录/文件 | 作用 |
| --- | --- |
| `cmd/server/main.go` | 程序入口：加载配置、注册模型、可选 seed、启动 Fiber。 |
| `config.yaml` | 本地运行配置；真实密钥不要提交到仓库。 |
| `internal/api` | 业务路由注册，是接口契约入口。 |
| `internal/model` | GORM 模型、JSON 投影、简单 hook 和枚举。 |
| `internal/query` | GET query 参数、分页、详情过滤和公开数据边界。 |
| `internal/service` | 复杂业务：附近聚合、出摊生命周期、申请审核、反馈处理、订单状态机、分享图。 |
| `internal/conf` | `custom` 配置读取：分享域名、前端目录、商品/收藏数量限制。 |
| `internal/bootstrap` | 服务注册、schema 准备、demo seed。 |
| `internal/contracttest` | 后端契约测试。 |
| `gkk` | 项目内置 gkk 框架代码。 |
| `API_GUIDE.md` | 给前端使用的接口文档和业务逻辑说明。 |

## 本地启动

前置条件：Go、PostgreSQL 可用，并准备好 `config.yaml` 中的数据库连接。

```bash
cd backend
go mod download
go run ./cmd/server
```

默认服务地址由 `server.port` 决定，当前本地配置为：

```text
http://localhost:8080
```

健康检查和页面代理由 Fiber/gkk 初始化统一处理；`custom.frontend` 配置后，Go 服务可以把前端构建产物代理到 `/`。

## 配置说明

核心配置结构：

```yaml
server:
  domain: "http://localhost:8080"
  port: "8080"
  cors:
    - "http://localhost:5173"

db:
  db: "postgres"
  name: "near"
  user: "postgres"
  password: "***"
  host: "127.0.0.1"
  port: "5432"

auth:
  mini:
    appid: "小程序 appid"
    key: "小程序 secret"
  appid: "公众号 appid"
  key: "公众号 secret"

oss:
  bucket: "bucket"
  domain: "https://oss.example.com"
  endpoint: "oss-cn-beijing.aliyuncs.com"
  id: "***"
  pwd: "***"

custom:
  share_url: "http://localhost:5173"
  seed_demo_data: true
  frontend: "../frontend/dist/build/h5"
  max_merchant_products: 50
  max_customer_favorites: 50
```

| 配置 | 说明 |
| --- | --- |
| `server.domain` | 后端公网域名，微信 OAuth、回调、分享能力会用到。 |
| `server.cors` | 允许访问 API 的前端域名。 |
| `db` | PostgreSQL 连接。 |
| `auth.mini` | 小程序 `wx.login` 换 token；缺少配置时不注册小程序登录路由。 |
| `auth.appid/key` | 公众号 OAuth、JS-SDK、扫码相关能力。 |
| `oss` | 上传文件和分享图存储。 |
| `custom.share_url` | 拼接 H5 分享链接：`{share_url}/share/{share_code}`。 |
| `custom.seed_demo_data` | 启动时填充演示数据。 |
| `custom.frontend` | 前端 H5 构建目录；为空则只提供 API。 |
| `custom.max_merchant_products` | 单商户商品数量上限，默认 50。 |
| `custom.max_customer_favorites` | 单顾客收藏数量上限，默认 50。 |

## 接口分组

| 分组 | 鉴权 | 说明 |
| --- | --- | --- |
| `POST /api/upload` | 前台 token | 文件上传，返回 OSS URL；支持出摊照 `1280x720`、商品图 `800x600`。 |
| `/api/pub/*` | 无 | 公开查询：附近摊位、商户详情、出摊状态、公开商品。 |
| `/api/customer/*` | 前台 token + `user_id` scope | 登录用户、入驻申请、反馈、收藏。 |
| `/api/merchant/*` | 前台 token + 商户身份 + `merchant_id` scope | 商户资料、出摊、商品、置顶。 |
| `/api/sys/*` | 后台 token + RBAC | 后台用户、菜单、角色、业务审核、商户、订单、反馈、活跃出摊。 |
| `/api/Wx/*` | 按能力区分 | 小程序登录、公众号 OAuth、JS-SDK、扫码、微信素材。 |

鉴权请求头统一使用：

```http
token: <jwt>
```

写接口成功通常返回 HTTP 200 空 body，前端不能强制按 JSON 解析成功响应。

完整接口契约见：

- `API_GUIDE.md`

## 核心业务结论

- 用户入口通过 `GET /api/customer/me` 的 `page_mode` 分流：`customer`、`application`、`merchant`。
- 公开附近列表只返回营业中、未过期、商户已启用且审核通过的摊位。
- 商户开始新出摊会关闭已有 active 出摊；到达 `expected_end_at` 后定时任务会自动结束出摊。
- 手动结束出摊会关闭该商户未接单订单并回补库存。
- 商户商品卡片摘要来自 `merchants.products`，商品创建/更新/删除/置顶/取消置顶会自动刷新。
- 单商户最多 3 个置顶商品；不足 3 个时用最新上架商品补足摘要。
- 分享码保存在商户表 `share_code`，小程序 scene 和 H5 分享链接都使用该值。
- `share_poster_url` 是固定分享图地址，`share_qrcode_url` 输出值与它一致。

## 常用命令

```bash
# 启动后端
cd backend && go run ./cmd/server

# 编译当前平台二进制
cd backend && go build -o server ./cmd/server

# 编译 Linux x86_64 二进制
cd backend && GOOS=linux GOARCH=amd64 go build -o ../release/stall-location-linux-amd64 ./cmd/server

# 运行后端测试
cd backend && go test ./...
```

## 部署方式

### 前后端分离

- 前端 H5/小程序使用 `VITE_API_BASE=https://后端域名`。
- 后端只提供 `/api`、`/api/Wx`、上传和后台接口。
- `server.cors` 必须包含前端域名。
- `custom.share_url` 设置为 H5 前端分享域名。

### Go 服务代理 H5 页面

1. 前端执行 H5 构建。
2. `custom.frontend` 指向 H5 构建目录。
3. Nginx 反向代理到 Go 服务。
4. Go 服务同时提供 `/api` 和 `/` 页面 fallback。

## 安全注意

- 不要把生产数据库密码、OSS 密钥、公众号/小程序 secret 写入公开仓库。
- 公开接口通过 DTO/`Json()` 投影隐藏手机号、用户 id、审核字段、owner 字段。
- 商户和顾客 owned 接口依赖 gkk scope 进入最终 SQL，前端不要传 `user_id` 或 `merchant_id` 来表达所属关系。

## 许可证与署名

本后端遵循根目录 `LICENSE` 的 `AGPL-3.0` 许可证。

- 保留版权声明和许可证声明。
- 网络服务修改版本需要向用户提供对应源码。
- 不得删除、隐藏、替换作者署名或冒名发布。
