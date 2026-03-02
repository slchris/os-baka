# OS-Baka

<p align="center">
  <strong>一个现代化的裸金属服务器自动化装机与资产管理平台</strong>
</p>

<p align="center">
  基于 PXE 网络启动 + iPXE 链式加载，提供从零开始的操作系统自动部署、全盘加密 (LUKS2 + TPM2)、DHCP 管理、密钥保管以及节点全生命周期管理。
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat-square&logo=react" alt="React" />
  <img src="https://img.shields.io/badge/TypeScript-5-3178C6?style=flat-square&logo=typescript" alt="TypeScript" />
  <img src="https://img.shields.io/badge/PostgreSQL-15-4169E1?style=flat-square&logo=postgresql" alt="PostgreSQL" />
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="MIT License" />
</p>

---

## 项目简介

**OS-Baka** 是面向数据中心 / 实验室环境的裸金属服务器批量装机与运维管理系统。通过 Web UI 即可完成"注册节点 → 配置 DHCP → 自动安装 OS → 加密磁盘 → 纳管"的全流程操作，消除手动安装操作系统和逐台配置的痛苦。

目前支持的操作系统：**Ubuntu** 和 **Debian**（基于 Preseed 无人值守安装）。

### 核心理念

- **零接触部署 (Zero-Touch Provisioning)**: 节点上电后自动 PXE 引导、安装 OS、配置加密，无需人工干预
- **安全优先**: 内置 LUKS2 / TPM2 全盘加密支持与 HashiCorp Vault 密钥托管
- **声明式管理**: 通过 Web UI / API 声明节点期望状态，系统自动完成 Preseed 生成
- **单一管控面**: 仪表盘 + WebShell + Key Vault 集中管理所有节点

---

## 功能清单

> **图例**: [x] 已实现 | [~] 部分实现 | [ ] 未实现

### 节点 / 资产管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 节点 CRUD | [x] | 创建、编辑、删除、列表查看节点 |
| CSV 批量导入节点 | [x] | 上传 CSV 文件批量注册 MAC / IP / Hostname |
| CSV 模板下载 | [x] | 一键下载标准 CSV 导入模板 |
| 节点状态管理 | [x] | 支持 PENDING / INSTALLING / ACTIVE / OFFLINE / ERROR / REBUILDING 状态 |
| 节点重建 (Rebuild) | [x] | 将节点标记为 `installing`，下次 PXE 启动自动重装 |
| 节点搜索 / 过滤 | [x] | 全局搜索栏 + 按状态过滤 |
| 资产标签 (Asset Tag) | [x] | 为节点绑定自定义资产编号 |
| 节点详情浮窗 (Tooltip) | [x] | 鼠标悬停展示节点详细信息 |
| AI 智能 CSV 分析 | [x] | 集成 Google Gemini API 对导入数据进行格式校验与风险分析 |
| 节点分组管理 | [x] | 按角色/机柜/环境分组，支持颜色标识，节点计数统计 |
| 节点标签 (Tags) | [x] | 为节点绑定 key-value 标签，支持批量替换 |
| 批量操作 | [x] | 批量重建、批量删除（含 DHCP/Tag 级联清理）、批量分组 |
| IPMI / BMC 带外管理 | [x] | 远程电源控制 (on/off/reset/cycle/status) + 连通性测试 |
| 节点心跳与健康检查 | [x] | Agent 上报 CPU/内存/磁盘/Uptime，后台自动检测离线节点 |

### PXE 网络启动

| 功能 | 状态 | 说明 |
|------|------|------|
| iPXE 链式加载 | [x] | 支持 BIOS (undionly.kpxe) 和 UEFI (ipxe.efi) 双模式启动 |
| 动态引导脚本生成 | [x] | 按节点 MAC 地址动态生成 iPXE boot script |
| Ubuntu / Debian Preseed | [x] | 自动生成 Preseed 无人值守安装配置 |
| Post-Install 脚本 | [x] | 安装完成后自动执行回调，更新节点状态 |
| 自定义镜像源 (Mirror URL) | [x] | 支持节点级 / 全局 / 环境变量三级镜像源优先级 |
| 自定义时区 | [x] | 每个节点可独立配置时区 (如 `Asia/Shanghai`) |
| SSH 服务配置 | [x] | 控制安装后是否开启 SSH 及是否允许 Root 登录 |
| 启动资产管理 (Boot Assets) | [x] | 上传 / 管理 Kernel / Initrd 等启动文件，SHA256 校验 |

### 安全与加密

| 功能 | 状态 | 说明 |
|------|------|------|
| LUKS2 全盘加密 | [x] | 安装时自动配置 LUKS2 (AES-XTS-PLAIN64) 加密 |
| TPM2 自动解锁 | [x] | 支持 TPM2 + PCR 绑定自动解锁磁盘 |
| PCR 策略自定义 | [x] | 可选绑定 PCR 0, 1, 2, 4, 7 等寄存器 |
| USB Keyfile 支持 | [x] | 支持 USB 物理密钥解锁方案 |
| Key Vault 管理页面 | [x] | 可视化管理各节点加密密钥槽位、生成 / 下载恢复密钥 |
| HashiCorp Vault 集成 | [x] | 支持将 LUKS 密码托管至 Vault KV v2，避免明文存库 |
| 数据库密码回退存储 | [x] | Vault 不可用时自动回退至数据库存储 (带日志告警) |
| 密钥轮换 (Key Rotation) | [x] | 后端安全生成新密码 (crypto/rand)，存入 Vault/DB，记录审计日志 |
| RBAC 细粒度权限 | [x] | JWT role claim + RequireRole 中间件，Admin/Operator/Auditor 三级权限矩阵 |
| API 速率限制 | [x] | Token Bucket 限流，登录端点 5 次/分钟防暴力破解 |
| API Key 认证 | [x] | Service Account 令牌 (osbaka\_xxx)，SHA-256 hash 存储，CRUD + 轮换 |

### DHCP 管理

| 功能 | 状态 | 说明 |
|------|------|------|
| DHCP 配置管理 | [x] | 创建/编辑/删除/激活 DHCP 配置档 |
| 多配置档支持 | [x] | 可创建多套 DHCP 配置，切换激活 |
| 静态地址保留 | [x] | MAC → IP 映射，支持手动创建 / 编辑 / 删除 |
| 从节点同步保留 | [x] | 一键将已注册节点的 MAC/IP 同步为 DHCP 保留 |
| dnsmasq 配置自动生成 | [x] | 根据 DB 配置自动生成 `/etc/dnsmasq.d/*.conf` |
| 网络接口自动发现 | [x] | 后端 API 自动列出系统可用网络接口 |
| PXE 启用/禁用开关 | [x] | DHCP 配置中可独立控制 PXE 功能 |
| DHCP 服务重启 | [x] | 配置变更后可一键重新生成配置并触发 dnsmasq 重载 |

### 仪表盘与监控

| 功能 | 状态 | 说明 |
|------|------|------|
| 概览仪表盘 | [x] | 显示节点总数、在线、加密、异常统计卡片 |
| 最近活动列表 | [x] | 展示最近操作审计日志 |
| 系统健康状态 | [x] | 真实展示后端服务、dnsmasq 进程状态、Vault 后端类型、用户数 |
| 通知系统 | [x] | 支持通知列表查看、标记已读、未读计数 |
| 审计日志 | [x] | 中间件自动记录变更操作，分页查询 + 条件过滤 |
| 节点心跳监控 | [x] | 展示 CPU / 内存 / 磁盘使用率、Uptime，自动标记离线 |

### 运维工具

| 功能 | 状态 | 说明 |
|------|------|------|
| WebShell 终端 | [x] | 前端 xterm.js + 后端 WebSocket SSH 代理，支持密码 / 密钥认证 |
| 用户管理 | [x] | 用户 CRUD、角色变更、密码修改、管理员权限检查 |
| 系统文档页面 | [x] | 内置架构说明、加密方案、恢复流程等技术文档 |
| Swagger API 文档 | [x] | 自动生成 OpenAPI 文档，含所有接口定义 |
| 健康检查端点 | [x] | `GET /api/v1/ping` 返回服务状态 |

### 用户体验

| 功能 | 状态 | 说明 |
|------|------|------|
| 暗色 / 亮色主题切换 | [x] | 支持手动切换 + 跟随系统偏好 |
| 响应式侧边栏导航 | [x] | 固定侧栏 + 面包屑导航 |
| JWT 认证 | [x] | Token-based 认证，自动过期跳转登录 |
| 用户菜单 | [x] | 右上角用户信息、Profile 入口、退出登录 |
| 表单验证 | [x] | MAC / IP 格式校验、必填字段检查 |

### DevOps / 部署

| 功能 | 状态 | 说明 |
|------|------|------|
| Docker Compose 一键部署 | [x] | 开发环境 (`docker-compose.yml`) + 生产环境 (`docker-compose.prod.yml`) |
| CI/CD Pipeline | [x] | GitHub Actions: 安全扫描 → 构建测试 → Docker 镜像构建 |
| 安全扫描 | [x] | GolangCI-Lint + Gosec + govulncheck + Trivy 多层安全审查 |
| Helm Chart / K8s 部署 | [ ] | 暂不支持 |
| Terraform Provider | [ ] | 暂未开发 |

---

## 架构概览

```
                    ┌───────────────────────────────────────────────┐
                    │                 OS-Baka 管理节点                │
                    │                                               │
                    │  ┌──────────┐  ┌──────────┐  ┌─────────────┐ │
                    │  │ Frontend │  │ Backend  │  │  PXE Service │ │
                    │  │ React 19 │  │ Go + Gin │  │  dnsmasq +   │ │
                    │  │ :3000    │  │ :8000    │  │  TFTP + iPXE │ │
                    │  └────┬─────┘  └────┬─────┘  └──────┬──────┘ │
                    │       │             │               │        │
                    │       │        ┌────┴────┐          │        │
                    │       └───────►│ API /v1 │◄─────────┘        │
                    │                └────┬────┘                   │
                    │                     │                        │
                    │           ┌─────────┴──────────┐             │
                    │           │                    │             │
                    │      ┌────▼────┐        ┌──────▼──────┐     │
                    │      │ Postgres│        │ Vault (可选) │     │
                    │      │  :5432  │        │   :8200     │     │
                    │      └─────────┘        └─────────────┘     │
                    └──────────────────────┬────────────────────────┘
                                           │
                    ───────── 网络 (DHCP + TFTP + HTTP) ──────────
                                           │
              ┌───────────────┬────────────┴─────────────┬───────────────┐
              │               │                          │               │
         ┌────▼────┐     ┌────▼────┐               ┌────▼────┐    ┌────▼────┐
         │ Node 1  │     │ Node 2  │     ...       │ Node N  │    │ Node M  │
         │ PXE Boot│     │ PXE Boot│               │ PXE Boot│    │ PXE Boot│
         │ LUKS+TPM│     │ LUKS+TPM│               │ LUKS+TPM│    │ LUKS    │
         └─────────┘     └─────────┘               └─────────┘    └─────────┘
```

### PXE 启动流程

```
目标节点上电
    │
    ▼
DHCP 请求 ──► dnsmasq 分配 IP + 引导文件
    │
    ▼
加载 iPXE 固件 (BIOS: undionly.kpxe / UEFI: ipxe.efi)
    │
    ▼
iPXE 链式加载 ──► GET /api/v1/pxe/init
    │
    ▼
动态引导脚本 ──► GET /api/v1/pxe/boot/{mac}
    │
    ▼
下载 Kernel + Initrd ──► 启动安装程序
    │
    ▼
Preseed 无人值守安装 ──► GET /api/v1/pxe/preseed/{mac}
    │
    ▼
自动安装 OS + LUKS 加密 + TPM2 注册
    │
    ▼
POST /api/v1/internal/nodes/{id}/status ──► 状态更新为 ACTIVE
```

---

## 快速开始

### 方式一：Docker Compose（推荐）

```bash
cd docker
docker-compose up -d
```

访问地址：
- **前端 UI**: http://localhost:3000
- **后端 API**: http://localhost:8000
- **API 文档**: http://localhost:8000/api/v1/docs/index.html

**启用 Vault 密钥管理（可选）：**
```bash
docker compose --profile vault up -d
```

### 方式二：本地开发

**终端 1 - 启动后端**:
```bash
cd backend
go mod download
go run ./cmd/server
```

**终端 2 - 启动前端**:
```bash
cd frontend
npm install
npm run dev
```

> 后端默认连接 PostgreSQL (`postgresql://osbaka:password@localhost:5432/osbaka`)，
> 请确保本地有 PostgreSQL 实例或通过 Docker 启动数据库：
> ```bash
> cd docker && docker-compose up -d db
> ```

---

## 默认登录

- **用户名**: `admin`
- **密码**: `admin`

> **注意**：生产环境请务必通过 `ADMIN_PASSWORD` 环境变量设置管理员密码。

---

## 项目结构

```
os-baka/
├── frontend/                 # React 19 + TypeScript + Vite 前端
│   └── src/
│       ├── pages/            # 页面组件 (Dashboard, Nodes, KeyVault, WebShell, AuditLogs, ...)
│       ├── components/       # 通用组件 (Sidebar, NodeTable, NodeTooltip, ...)
│       ├── services/         # API 客户端 & 业务服务层
│       │   └── __tests__/    # 前端测试 (vitest)
│       ├── router.tsx        # 路由配置 (受保护路由)
│       └── types.ts          # TypeScript 类型定义
├── backend/                  # Go (Gin + GORM + PostgreSQL) 后端
│   ├── cmd/server/           # 应用入口 (main.go)
│   ├── internal/
│   │   ├── api/              # HTTP Handler 层
│   │   │   ├── auth.go       # 认证 (Login / Me)
│   │   │   ├── nodes.go      # 节点 CRUD + 状态管理 + 密钥轮换
│   │   │   ├── users.go      # 用户管理 CRUD
│   │   │   ├── pxe.go        # PXE 脚本生成 (init/boot/preseed/postinstall)
│   │   │   ├── dhcp.go       # DHCP 配置 & 保留管理
│   │   │   ├── dnsmasq.go    # dnsmasq 配置文件自动生成
│   │   │   ├── asset.go      # 启动资产上传 / 管理
│   │   │   ├── dashboard.go  # 仪表盘数据聚合
│   │   │   ├── audit.go      # 审计日志 (中间件 + 查询)
│   │   │   ├── ipmi.go       # IPMI 电源管理
│   │   │   ├── groups.go     # 节点分组 & 标签管理
│   │   │   ├── bulk.go       # 批量操作 (重建/删除/分组)
│   │   │   ├── heartbeat.go  # 节点心跳 & 离线检测
│   │   │   ├── apikeys.go    # API Key 管理 (CRUD + 轮换)
│   │   │   ├── ratelimit.go  # 速率限制中间件
│   │   │   ├── websocket.go  # WebSocket SSH 代理
│   │   │   ├── utils.go      # JWT 生成/验证 + RBAC 中间件
│   │   │   └── notifications.go # 通知管理
│   │   ├── config/           # 配置加载 (YAML + 环境变量)
│   │   ├── model/            # 数据模型 (GORM) & 数据库初始化
│   │   ├── vault/            # 密钥存储抽象层 (Vault / DB 双后端)
│   │   └── sysutil/          # 系统工具 (网络接口探测等)
│   └── docs/                 # Swagger 自动生成文档
├── pxe-services/             # PXE 引导服务
│   ├── dnsmasq.conf          # dnsmasq 基础配置
│   ├── boot.ipxe             # iPXE 链式加载脚本
│   ├── ipxe/                 # iPXE 固件 (BIOS + UEFI)
│   ├── Dockerfile            # PXE 服务容器构建
│   └── start.sh              # 容器启动脚本
├── docker/                   # Docker Compose 编排
│   ├── docker-compose.yml    # 开发环境
│   ├── docker-compose.prod.yml # 生产环境 (health check + resource limits)
│   └── .env.example          # 环境变量模板
├── scripts/                  # 部署脚本
│   └── deploy.sh             # 自动化部署脚本
├── .github/workflows/        # CI/CD
│   └── ci.yml                # 安全扫描 + 构建 + 测试 + Docker 镜像
└── config.yaml               # 统一配置文件
```

---

## 技术栈

| 层次 | 技术 |
|------|------|
| **前端** | React 19 + TypeScript 5 + Vite + Tailwind CSS v4 + React Router v7 |
| **UI 组件** | Lucide Icons + xterm.js (WebShell) |
| **AI 集成** | Google Gemini API (CSV 数据分析, 可选) |
| **后端** | Go 1.24 + Gin + GORM |
| **数据库** | PostgreSQL 15 |
| **密钥管理** | HashiCorp Vault KV v2 (可选) + 数据库回退 |
| **PXE 服务** | dnsmasq (DHCP / TFTP) + iPXE 链式加载 |
| **API 文档** | Swagger / OpenAPI (swaggo) |
| **CI/CD** | GitHub Actions |
| **安全扫描** | GolangCI-Lint, Gosec, govulncheck, Trivy |
| **部署** | Docker + Docker Compose |

---

## API 概览

启动后端后，完整 API 文档请访问 Swagger UI：  
**http://localhost:8000/api/v1/docs/index.html**

### 主要端点

| 分组 | 方法 | 路径 | 说明 |
|------|------|------|------|
| **认证** | POST | `/api/v1/auth/login` | 用户登录，获取 JWT (登录限流保护) |
| | GET | `/api/v1/auth/me` | 获取当前用户信息 |
| **节点** | GET | `/api/v1/nodes` | 列出所有节点 |
| | POST | `/api/v1/nodes` | 创建节点 (admin, operator) |
| | PUT | `/api/v1/nodes/:id` | 更新节点 (admin, operator) |
| | DELETE | `/api/v1/nodes/:id` | 删除节点 (admin) |
| | POST | `/api/v1/nodes/:id/rebuild` | 重建节点 (admin, operator) |
| | GET | `/api/v1/nodes/:id/passphrase` | 获取节点加密密码 (admin) |
| | POST | `/api/v1/nodes/:id/rotate-passphrase` | 轮换加密密码 (admin) |
| | POST | `/api/v1/nodes/:id/power` | IPMI 电源操作 (admin, operator) |
| | GET | `/api/v1/nodes/:id/ipmi/test` | 测试 IPMI 连通性 (admin, operator) |
| | PUT | `/api/v1/nodes/:id/group` | 分配节点到分组 (admin, operator) |
| | GET/PUT | `/api/v1/nodes/:id/tags` | 节点标签管理 |
| **批量操作** | POST | `/api/v1/nodes/bulk/rebuild` | 批量重建 (admin, operator) |
| | POST | `/api/v1/nodes/bulk/delete` | 批量删除 (admin) |
| | PUT | `/api/v1/nodes/bulk/group` | 批量分组 (admin, operator) |
| **分组** | GET/POST | `/api/v1/groups` | 节点分组管理 |
| | PUT/DELETE | `/api/v1/groups/:id` | 分组更新/删除 (admin) |
| **用户** | GET | `/api/v1/users` | 列出用户 |
| | POST | `/api/v1/users` | 创建用户 |
| | PUT | `/api/v1/users/:id` | 更新用户 |
| | DELETE | `/api/v1/users/:id` | 删除用户 |
| **PXE** | GET | `/api/v1/pxe/init` | iPXE 初始化脚本 |
| | GET | `/api/v1/pxe/boot/:mac` | 动态启动脚本 |
| | GET | `/api/v1/pxe/preseed/:mac` | Preseed 配置 |
| | GET | `/api/v1/pxe/postinstall/:mac` | 安装后脚本 |
| **DHCP** | GET/POST | `/api/v1/dhcp/configs` | DHCP 配置管理 |
| | GET | `/api/v1/dhcp/config/active` | 获取当前活跃配置 |
| | POST | `/api/v1/dhcp/service/restart` | 重启 DHCP 服务 |
| | GET/POST | `/api/v1/dhcp/reservations` | 静态地址保留管理 |
| | POST | `/api/v1/dhcp/reservations/sync` | 从节点同步保留 |
| **审计** | GET | `/api/v1/audit-logs` | 审计日志分页查询 |
| **API Key** | GET/POST | `/api/v1/api-keys` | API Key 管理 (admin) |
| | DELETE | `/api/v1/api-keys/:id` | 撤销 Key (admin) |
| | POST | `/api/v1/api-keys/:id/rotate` | 轮换 Key (admin) |
| **资产** | GET/POST | `/api/v1/assets/boot` | 启动资产管理 |
| **仪表盘** | GET | `/api/v1/dashboard/summary` | 概览统计数据 |
| **通知** | GET | `/api/v1/notifications` | 通知列表 |
| **WebSocket** | GET | `/api/v1/ws/ssh` | WebShell SSH 代理 |
| **内部** | POST | `/api/v1/internal/nodes/:id/heartbeat` | 节点心跳上报 |
| **系统** | GET | `/api/v1/system/interfaces` | 网络接口列表 |

---

## 配置

主要配置文件为 `config.yaml`，也支持环境变量覆盖：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `DATABASE_URL` | PostgreSQL 连接串 | `postgresql://osbaka:password@localhost:5432/osbaka` |
| `PORT` | 后端监听端口 | `8000` |
| `SECRET_KEY` | JWT 签名密钥 | `change-this-in-production` |
| `GIN_MODE` | Gin 运行模式 | `debug` |
| `ADMIN_PASSWORD` | 管理员初始密码 | `admin` |
| `EXTERNAL_IP` | PXE 服务对外 IP | 自动检测 |
| `VITE_API_URL` | 前端 API 地址 | `http://localhost:8000/api/v1` |
| `VITE_GEMINI_API_KEY` | Gemini AI API Key (可选) | - |
| `VAULT_ENABLED` | 启用 Vault 密钥管理 | `false` |
| `VAULT_ADDR` | Vault 服务地址 | `http://vault:8200` |
| `VAULT_TOKEN` | Vault 认证 Token | - |
| `PXE_ASSETS_DIR` | 启动资产存储目录 | `/tftpboot` |
| `PXE_MIRROR_URL` | 全局 OS 镜像源 | 各发行版默认源 |

---

## 开发指南

### 后端开发

```bash
cd backend
go mod download

# 运行
go run ./cmd/server

# 测试
go test -v ./...

# Lint
golangci-lint run --timeout=5m

# 更新 Swagger 文档
swag init -g cmd/server/main.go
```

### 前端开发

```bash
cd frontend
npm install

# 开发服务器
npm run dev

# 类型检查
npx tsc --noEmit

# 测试
npm test

# 构建
npm run build
```

---

## 许可证

[MIT License](LICENSE)
