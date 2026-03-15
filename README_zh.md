# YoBFF

[简体中文](./README_zh.md) | [English](./README.md)

YoBFF 是一个面向云原生出口场景的 BFF 网关与负载均衡项目，采用 Go 实现数据面与控制面，并内置 React 管理台。

## 当前已实现能力

### 1. 架构与部署
- 控制面与数据面在同一进程内解耦运行，统一生命周期管理。
- 支持将管理台静态资源通过 Go embed 打包进二进制，也支持通过 `ADMIN_UI_DIR` 使用外部静态目录。
- 支持单二进制部署，状态数据持久化到 SQLite（`modernc.org/sqlite`，无 CGO）。

### 2. 流量治理与网关能力
- 支持基于 Host 的路由转发（精确域名与通配符域名）。
- 支持负载均衡池与路由规则管理（控制面 API + 管理台）。
- 支持 HTTP/HTTPS 转发，支持 SNI 证书匹配与证书管理。
- 支持 `X-Forwarded-For` 解析与受信代理 CIDR 校验。

### 3. 安全与访问控制
- 管理 API 基于 Bearer Token 鉴权。
- 支持登录失败后验证码校验（防暴力破解）。
- 支持请求级 `X-Request-ID` 追踪与固定窗口限流。
- 支持 HSTS 响应头开关（HTTPS 请求场景）。

### 4. 配置治理与审计
- 支持配置读取、预检、应用、热重载。
- 支持全局配置版本快照、版本查询与回滚。
- 支持站点级配置版本、回滚、差异比对与日志流配置。
- 审计日志支持动作/操作人/目标/时间筛选与分页查询。
- 审计操作人默认绑定登录用户身份，并兼容 `X-Operator`。

### 5. CDN 与源站防护
- 内置 Cloudflare、阿里云 ESA、腾讯云 TEO 插件。
- 支持定时同步 CDN 出口网段（CIDR）并更新访问放行策略。
- 支持站点维度的 CDN 回源同步状态查询与手动刷新。

### 6. 管理台模块
- 已提供：仪表盘、流量管理、证书管理、观测中心、系统设置、可视化实验室（Weaver）。
- 观测中心支持审计日志筛选与分页交互。
- Weaver 支持草稿创建、编辑、查询、运行及对应审计记录。

## API 契约

- 控制面 OpenAPI 契约位于：`contracts/openapi/admin.yaml`。
- 已覆盖登录、配置、流量池/路由、站点管理、证书管理、日志统计、审计日志、Weaver 等接口。

## 快速开始

### 环境要求
- Go 1.24 或更高版本
- Node.js（仅前端开发需要）
- pnpm（仅前端开发需要）

### 启动后端
1. 克隆仓库。
2. 初始化环境变量：`cp .env.example .env`（Windows 可改为等效复制命令）。
3. 启动服务：

```bash
go run .
```

默认访问地址：
- 管理台：`http://localhost:8080/admin`
- 健康检查：`http://localhost:8080/healthz`

### 前端开发
1. 进入前端目录：

```bash
cd web/source
pnpm install
pnpm dev
```

## 常用验证命令

- 后端测试：`go test ./...`
- 后端静态检查：`go vet ./...`
- 前端检查：`pnpm lint`
- 前端构建：`pnpm build`

## Feature（规划中）

以下内容尚未完成，当前列为规划方向：
- 动态服务发现：Kubernetes Service / Consul 对接。
- 应用层鉴权插件：JWT 校验与 OIDC 代理。
- 可观测性扩展：Prometheus 指标导出与 Grafana 看板模板。
- 多实例配置同步：基于 Etcd 的一致性同步。
- 高级 L7 策略：声明式 Rewrite、熔断等能力。

## 许可证

Copyright 2026 YoBFF.

本项目采用 Apache License 2.0 许可证。详情请参阅 [LICENSE](LICENSE) 文件。
