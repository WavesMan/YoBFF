# YoBFF

[简体中文](./README_zh.md) | [English](./README.md)

YoBFF is a lightweight, high-performance BFF (Backend For Frontend) gateway and load balancer designed to serve as a secure exit point for cluster services. It prioritizes operational transparency and security, providing a built-in management interface, automated origin protection, and atomic configuration rollbacks.

## Project Vision

Traditional gateways often separate the data plane from the management interface or require complex external dependencies. YoBFF is built as a single, independent binary that encapsulates both the high-performance Go-based forwarding engine and a modern React-based control plane. It is specifically designed for cloud-native environments where origin IP leakage is a concern.

## Core Capabilities

### 1. Architectural Integrity
- Separation of Control Plane and Data Plane within a unified lifecycle.
- Single-binary deployment with embedded UI assets using Go embed.
- CGO-free implementation utilizing pure Go SQLite for state persistence.
- Atomic configuration updates via memory snapshots to ensure zero-downtime reloads.

### 2. Traffic Management & Load Balancing
- Host-based routing supporting exact matches and wildcard domains.
- Weighted Round-Robin (WRR) load balancing with health-aware node management.
- Protocol support for HTTP and HTTPS (including SNI matching).
- Primary/Fallback pool logic for high availability at the site level.

### 3. Automated Origin Protection
- Integrated plugins for Cloudflare, Aliyun ESA, and Tencent TEO.
- Automated synchronization of cloud provider egress CIDRs to prevent unauthorized direct-to-ip access.
- Custom 403 error page rendering with request tracking identifiers for rapid troubleshooting.

### 4. Enterprise-Grade Configuration Control
- Git-style versioning: Every configuration change is captured as a snapshot in the database.
- Atomic Rollback: Revert to any historical configuration state via the UI with a single click.
- AES-GCM Encryption: Sensitive data such as CDN API keys are encrypted at rest and never echoed in the UI or API responses.
- Audit Logging: Comprehensive tracking of all administrative actions.

## Security Implementation

- Authentication: Bearer Token-based access control for all management APIs.
- Brute-force Protection: Built-in login guard with fail-count aware Captcha requirements.
- Edge Security: Support for HSTS and trusted proxy headers (X-Forwarded-For) with CIDR validation.
- Privacy: Automated redaction of secrets in diffs and logs.

## Quick Start

### Prerequisites
- Go 1.22 or higher
- Node.js (for frontend development only)

### Backend Execution
1. Clone the repository.
2. Initialize environment: `cp .env.example .env`.
3. Run the gateway:
```bash
go run .
```
The management interface is accessible by default at `http://localhost:8080/admin`.

### Frontend Development
1. Navigate to the source directory:
```bash
cd web/source
pnpm install
pnpm dev
```

## Technical Specification

- Backend: Go (Golang)
- Frontend: React 19, TypeScript, Vite
- Storage: SQLite (modernc.org/sqlite)
- Logging: Asynchronous non-blocking pipeline using Uber-Zap

## Roadmap (Planned Features)

The following features are currently under consideration or in early development:

- Dynamic Service Discovery: Native integration with Kubernetes Service APIs and Consul.
- Application-Level Plugins: Support for JWT validation and OIDC proxying.
- Observability: Prometheus metrics exporter and Grafana dashboard templates.
- Cluster Sync: Configuration synchronization across multiple YoBFF instances using Etcd.
- Advanced L7 Policies: Declarative request rewriting and circuit breaking.

## License

Copyright 2026 YoBFF.

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0.