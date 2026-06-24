# 出摊啦 frontend

当前 `frontend` 是「出摊啦」uni-app 前端项目，主要面向微信小程序，同时支持 H5 本地调试。

> 说明：原 `frontend-uni` 已覆盖合并到当前 `frontend` 目录。后续请以 `frontend` 为唯一前端维护目录。

## 快速开始

```bash
cd frontend
npm install
npm run dev:mp-weixin
```

微信开发者工具导入：

```text
frontend/dist/dev/mp-weixin
```

生产构建：

```bash
cd frontend
npm run build:mp-weixin
```

微信开发者工具导入：

```text
frontend/dist/build/mp-weixin
```

H5 调试：

```bash
cd frontend
npm run dev:h5
```

## 文档

- [项目介绍](./docs/PROJECT_OVERVIEW.md)
- [使用文档](./docs/USAGE.md)

## 核心配置
- 页面入口：`src/pages/customer/index.vue`
- 小程序配置：`src/manifest.json`
- 页面配置：`src/pages.json`

## npm 脚本

```bash
npm run dev:h5          # H5 本地调试
npm run dev:mp-weixin   # 微信小程序开发构建
npm run build:h5        # H5 生产构建
npm run build:mp-weixin # 微信小程序生产构建
```

## 功能范围

- 顾客端：附近商户列表、地图、商户详情、商品、收藏、导航、入驻申请、反馈。
- 商户端：申请状态、商户工作台、开始/结束出摊、商品管理、置顶商品、分享二维码。

更多目录说明、接口说明、定位/登录/构建注意事项见 `docs/`。

## 许可证与署名

本前端遵循根目录 `LICENSE` 的 `AGPL-3.0` 许可证。

- 保留版权声明和许可证声明。
- 网络服务修改版本需要向用户提供对应源码。
- 不得删除、隐藏、替换作者署名或冒名发布。
