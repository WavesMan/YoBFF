# YoBFF

YoBFF 是一个以 Go 为核心的 BFF 负载均衡网关，配套内置管理端与配置热重载能力，适合用来承载多上游的统一入口和观测面。

## 主要能力
- 控制面与数据面统一编排，支持 HTTP 与 HTTPS
- 管理端界面与 API 合并在 /admin 路径
- 配置热重载与版本回滚
- 站点管理、证书管理与站点日志流探查
- 基础日志统计与告警提示
- CDN 同步能力（插件式）

## 快速开始

### 运行后端服务
1. 准备环境
   - Go 1.22
2. 配置环境变量
   - 复制 .env.example 为 .env 并按需修改
3. 启动服务

```bash
go run .
```

默认监听地址来自 config/config.json，并可被 .env 中的环境变量覆盖。默认情况下管理端入口为：
```
http://localhost:8080/admin
```

### 开发前端管理端
1. 进入前端目录并安装依赖

```bash
cd web/source
pnpm install
```

2. 启动开发服务器

```bash
pnpm dev
```

### 构建并接入管理端
前端构建产物默认输出到 web/source/dist。你有两种方式接入：

- 运行时挂载：在 .env 中设置 ADMIN_UI_DIR 指向构建产物目录，例如 web/source/dist
- 内嵌构建：将构建产物同步到 web/ui 后再编译 Go 服务

```bash
cd web/source
pnpm build
```

## 配置说明
- 服务配置：config/config.json
- 环境变量示例：.env.example
- 管理端监听配置已废弃，管理接口统一使用 /admin 路径

## API 契约
管理端 API 的 OpenAPI 文件位于：
- contracts/openapi/admin.yaml

## 测试与质量
后端测试：

```bash
go test ./...
```

前端检查：

```bash
cd web/source
pnpm run lint
pnpm exec tsc -b
```

## 目录结构
- internal：后端核心逻辑
- contracts/openapi：接口契约
- config：默认配置与数据库目录
- web/source：前端管理端源码
- web/ui：内嵌管理端静态资源
