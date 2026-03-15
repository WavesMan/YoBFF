# YoBFF

[简体中文](./README_zh.md) | [English](./README.md)

YoBFF is a cloud-native BFF gateway and load balancer. It provides a Go-based data/control plane with a built-in React admin console.

## What Is Implemented Today

### 1. Architecture & Deployment
- Control plane and data plane run in one process with clear separation.
- Admin UI can be served from embedded assets (Go embed) or an external directory via `ADMIN_UI_DIR`.
- Single-binary deployment is supported, with state persisted in SQLite (`modernc.org/sqlite`, CGO-free).

### 2. Gateway & Traffic Management
- Host-based routing with exact and wildcard domain support.
- Load balancer pools and route rules are manageable via API and admin UI.
- HTTP/HTTPS forwarding with SNI certificate matching and certificate management.
- `X-Forwarded-For` processing with trusted proxy CIDR validation.

### 3. Security & Access Control
- Bearer token authentication for admin APIs.
- Brute-force mitigation via captcha requirement after failed login attempts.
- Request tracing with `X-Request-ID` and fixed-window rate limiting.
- Configurable HSTS response header for HTTPS requests.

### 4. Config Governance & Auditability
- Config read/validate/apply/reload workflows.
- Global config snapshot history, version listing, and rollback.
- Site-level config versions, rollback, diff, and log stream settings.
- Audit logs with filtering by action/operator/target/time and paginated responses.
- Audit operator is bound to the authenticated login user, with `X-Operator` compatibility.

### 5. CDN & Origin Protection
- Built-in providers: Cloudflare, Aliyun ESA, Tencent TEO.
- Scheduled CIDR synchronization from CDN providers.
- Site-level CDN origin sync status query and manual refresh endpoints.

### 6. Admin UI Modules
- Available modules: Dashboard, Traffic, Certificates, Observability, System, Weaver.
- Observability includes filterable and paginated audit log interactions.
- Weaver supports draft create/update/list/detail/run and writes audit records.

## API Contract

- OpenAPI contract is located at: `contracts/openapi/admin.yaml`.
- It includes login, config, LB pools/routes, site management, certificates, log stats, audit logs, and Weaver APIs.

## Quick Start

### Prerequisites
- Go 1.24 or higher
- Node.js (frontend development only)
- pnpm (frontend development only)

### Run Backend
1. Clone the repository.
2. Initialize environment variables: `cp .env.example .env` (use equivalent copy command on Windows).
3. Start the service:

```bash
go run .
```

Default endpoints:
- Admin UI: `http://localhost:8080/admin`
- Health check: `http://localhost:8080/healthz`

### Frontend Development
1. Enter the frontend directory:

```bash
cd web/source
pnpm install
pnpm dev
```

## Common Verification Commands

- Backend tests: `go test ./...`
- Backend static checks: `go vet ./...`
- Frontend lint: `pnpm lint`
- Frontend build: `pnpm build`

## Features (Planned)

The following items are not completed yet:
- Dynamic service discovery via Kubernetes Service APIs and Consul.
- Application-level auth plugins (JWT validation and OIDC proxy).
- Observability extension (Prometheus exporter and Grafana dashboards).
- Multi-instance config synchronization using Etcd.
- Advanced L7 policies (declarative rewrite, circuit breaking, etc.).

## License

Copyright 2026 YoBFF.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.
