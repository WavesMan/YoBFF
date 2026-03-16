# YoBFF 控制面 API 契约说明（阶段一基线）

## 1. 文档目标

本文件定义 YoBFF 控制面 API 的统一契约基线，用于保证以下协作边界一致：

- 前后端接口调用一致
- 服务内外部集成接口行为一致
- 第三方对接数据结构与错误处理一致
- 自动化测试与 CI 校验规则一致

对应 OpenAPI 源文件：`contracts/openapi/admin.yaml`。

## 2. 契约范围

当前基线覆盖以下模块：

- 鉴权与登录：`/api/v1/login`、`/api/v1/logout`、`/api/v1/captcha`
- 配置治理：`/api/v1/config`、`/api/v1/config/validate`、`/api/v1/config/versions`、`/api/v1/config/rollback`
- 流量管理：`/api/v1/lb/pools`、`/api/v1/lb/routes`
- 站点管理：`/api/v1/sites`、`/api/v1/site-groups`、站点配置/日志/CDN 子路由
- 观测审计：`/api/v1/log/level`、`/api/v1/log/stats`、`/api/v1/audit/logs`
- Weaver 编排：`/api/v1/weaver/drafts`、`/api/v1/weaver/drafts/{draft_id}`、`/api/v1/weaver/drafts/{draft_id}/run`

## 3. 协议约定

- 协议：HTTP/JSON
- 编码：UTF-8
- 鉴权：`Authorization: Bearer <token>`
- 追踪：响应头必须回传 `X-Request-ID`
- 限流：响应头返回 `X-RateLimit-Limit`、`X-RateLimit-Remaining`、`X-RateLimit-Reset`
- 操作人：可通过 `X-Operator` 指定，缺省时回退登录用户

## 4. 请求与响应格式

### 4.1 成功响应

- 查询类接口返回资源对象或列表对象
- 写接口返回更新后的资源对象或状态对象
- 列表统一使用 `items` 字段承载集合

### 4.2 错误响应

统一错误结构：

- `error_code`: 机器可读错误码
- `message`: 可读错误信息
- `request_id`: 请求追踪标识

验证错误附加 `errors` 数组，元素为 `{ path, message }`。

## 5. 关键错误码清单

- `unauthorized`：鉴权失败
- `auth_not_configured`：服务未配置鉴权令牌
- `invalid_request`：请求体或参数不合法
- `draft_not_found`：Weaver 草稿不存在
- `weaver_create_failed`：Weaver 草稿创建失败
- `weaver_update_failed`：Weaver 草稿更新失败
- `weaver_run_failed`：Weaver 草稿运行失败
- `site_not_found`：站点不存在
- `version_not_found`：配置版本不存在
- `rate_limited`：触发限流

## 6. 版本策略

- 主版本策略：`/api/v1` 路径版本化
- 兼容变更：
  - 新增可选字段
  - 新增可选查询参数
  - 新增不影响旧逻辑的接口
- 非兼容变更：
  - 删除字段或修改字段语义
  - 修改必填项约束
  - 修改错误码语义

非兼容变更必须升级路径主版本（`v2`）并提供迁移说明。

## 7. 变更规则

- 契约优先：代码变更前先更新 OpenAPI 草案
- 双向对齐：后端路由与前端类型同步评审
- 变更留痕：每次变更必须附带评审记录与回归项
- 回滚约束：线上问题回滚优先保持契约可用性

## 8. 第三方集成契约边界

第三方集成主要在 CDN 供应商与证书链路：

- CDN 配置入口：`/api/v1/config/cdn`、`/api/v1/sites/{site_id}/cdn/origin/*`
- 对外暴露的配置字段仅包含必要连接信息
- 密钥类字段只允许写入，不允许在读取接口明文回显
- 第三方错误信息需映射为统一 `error_code` 与 `message`

## 9. 自动化校验规则

当前 CI 应至少执行以下校验：

- 契约关键路径存在性检查（含 Weaver 路由）
- 契约关键 Schema 存在性检查（含 Weaver 请求/响应）
- 后端接口集成测试（草稿创建、查询、更新、运行）
- 基础静态检查与单元测试

建议命令：

- `go test ./contracts/openapi -run TestAdminOpenAPIContractConsistency`
- `go test ./internal/admin -run TestWeaverDraftRoutes`
- `go test ./...`
- `go vet ./...`

## 10. 追溯关系

- 计划基线：`feature-docs/0316/0316-00_Feature_Dev.md`
- 阶段拆解：`feature-docs/0316/0316-01_阶段一任务拆解.md`
- 契约源文件：`contracts/openapi/admin.yaml`
- 契约评审记录：`feature-docs/0316/0316-05_契约评审记录.md`
- CI 校验配置：`.github/workflows/contract_consistency.yml`
