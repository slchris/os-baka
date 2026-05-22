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
- **安全优先**: 内置 LUKS2 / TPM2 全盘加密支持,LUKS 密码使用 AES-256-GCM 加密存储
- **声明式管理**: 通过 Web UI / API 声明节点期望状态，系统自动完成 Preseed 生成
- **单一管控面**: 仪表盘 + WebShell + Key Vault 集中管理所有节点

---

## 功能清单

> **图例**: [x] 已实现 | [~] 部分实现 | [ ] 未实现

### 节点 / 资产管理

| 功能 | 状态 | 说明 |
|------|------|------|
| 节点 CRUD | [x] | 创建、编辑、删除、列表查看节点;删除时把 hostname/MAC/IP 快照写入审计日志 |
| 节点自动发现 | [x] | 未知 MAC 触发 PXE 时自动创建 `discovered` 状态节点,显示在 UI 等待操作员审批;不会自动装机 |
| CSV 批量导入节点 | [~] | 前端解析 CSV 并逐行调用 POST /nodes(无专用后端批量端点) |
| CSV 模板下载 | [x] | 前端生成标准 CSV 导入模板 |
| 节点状态机 | [x] | `discovered / pending / installing / active / maintenance / error / offline` 七态,合法性转换检查 + 进入 installing 自动打时间戳;`discovered` 仅由 PXE 自动发现路径产生 |
| 节点重建 (Rebuild) | [x] | 将节点标记为 `installing` + **自动通过 IPMI 触发 power cycle**;`installing` 超时(默认 120 min)自动转 `error`。配置了 IPMI 的节点装机真正零接触 |
| 节点搜索 / 过滤 | [x] | 全局搜索栏 + 按状态过滤 |
| 资产标签 (Asset Tag) | [x] | 为节点绑定自定义资产编号 |
| 节点详情浮窗 (Tooltip) | [x] | 鼠标悬停展示节点详细信息 |
| AI 智能 CSV 分析 | [x] | 前端集成 Google Gemini API 对导入数据进行格式校验与风险分析(需 `VITE_GEMINI_API_KEY`) |
| 节点分组管理 | [x] | 按角色/机柜/环境分组,支持颜色标识,节点计数统计 |
| 节点标签 (Tags) | [x] | 为节点绑定 key-value 标签,支持批量替换 |
| 批量操作 | [x] | 批量重建、批量删除(含 DHCP/Tag 级联清理 + 逐节点删除快照)、批量分组 |
| IPMI / BMC 带外管理 | [x] | 远程电源控制 (on/off/reset/cycle/status) + 连通性测试 + IPMI 密码字段级加密 + Rebuild 自动 power cycle(后台并发,信号量限流) |
| 节点心跳与健康检查 | [x] | Agent 上报 CPU/内存/磁盘/Uptime;后台同一 ticker 跑两个检查器:`active`→`offline` 超时下线 + `installing`→`error` 装机超时 |

### PXE 网络启动

| 功能 | 状态 | 说明 |
|------|------|------|
| iPXE 链式加载 | [x] | 支持 BIOS (undionly.kpxe) 和 UEFI (ipxe.efi) 双模式启动 |
| 动态引导脚本生成 | [x] | 按节点 MAC 地址动态生成 iPXE boot script |
| Ubuntu / Debian Preseed | [x] | 自动生成 Preseed 无人值守安装配置 |
| 目标磁盘自动探测 | [x] | 节点 `target_disk` 字段为空时,preseed 在 d-i 阶段自动选盘(nvme* > vd* > sd*,按容量选最大,排除 USB / removable);填了则按填写值装机 |
| 一次性装机 token | [x] | Boot script 颁发短期 token,Preseed/PostInstall 校验,防止同网段攻击者拼 MAC 拉密码 |
| Post-Install 脚本 | [x] | 安装完成后自动执行回调,更新节点状态(走状态机 + token 消费) |
| 自定义镜像源 (Mirror URL) | [x] | 支持节点级 / 全局 / OS 专用 env / DB 配置 / 内置默认 五级镜像源优先级;**主仓和 security 仓分开配置**,适配内网典型布局 |
| 自定义时区 | [x] | 每个节点可独立配置时区 (如 `Asia/Shanghai`) |
| SSH 服务配置 | [x] | 控制安装后是否开启 SSH 及是否允许 Root 登录 |
| 启动资产管理 (Boot Assets) | [x] | 上传 / 管理 Kernel / Initrd 等启动文件,SHA256 校验 |
| dnsmasq 配置去抖调度 | [x] | 批量节点操作下 dnsmasq 配置 leading+trailing 合并,N 次写最多触发 2 次重生成 |

### 安全与加密

| 功能 | 状态 | 说明 |
|------|------|------|
| LUKS2 全盘加密 | [x] | 安装时自动配置 LUKS2 (AES-XTS-PLAIN64) 加密 |
| TPM2 自动解锁 | [x] | 支持 TPM2 + PCR 绑定自动解锁磁盘 |
| PCR 策略自定义 | [x] | 可选绑定 PCR 0, 1, 2, 4, 7 等寄存器 |
| USB Keyfile 支持 | [x] | 支持 USB 物理密钥解锁方案 |
| Key Vault 管理页面 | [~] | UI 提供"槽位"视图;后端目前只存单个 LUKS passphrase(支持获取与轮换),槽位划分是前端表达 |
| 字段级加密存储 | [x] | LUKS 密码、IPMI 密码、Root 密码使用 AES-256-GCM 加密落库 (`FIELD_ENCRYPTION_KEY`),不设密钥则降级为明文 + 启动 warn |
| 密钥轮换 (Key Rotation) | [x] | 后端安全生成新密码 (crypto/rand),加密落库,记录审计日志 |
| RBAC 细粒度权限 | [x] | JWT role claim + RequireRole 中间件,Admin/Operator/Auditor 三级权限矩阵 |
| API 速率限制 | [x] | Token Bucket 限流,登录端点 5 次/分钟防暴力破解 |
| API Key 认证 | [x] | Service Account 令牌 (`osbaka_xxx`),SHA-256 hash 存储,CRUD + 轮换 |

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
| 系统健康状态 | [x] | 展示后端服务、dnsmasq 进程状态、dnsmasq 调度器(last_run / last_error / run_count)、用户数 |
| 通知系统 | [x] | 支持通知列表查看、标记已读、未读计数 |
| 审计日志 | [x] | 中间件自动记录变更操作 + 关键路径(状态机/删除/电源/Key 轮换)显式记录;**同步写入**(失败可见,非异步丢弃);**节点删除携带 JSON 快照**保留 hostname/MAC/IP/资产标签 |
| 节点心跳监控 | [x] | 展示 CPU / 内存 / 磁盘使用率、Uptime,只允许从 `active`/`offline` 进入 `active`(不影响 maintenance/installing 节点) |

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
| 表单验证 | [x] | MAC / IP / Hostname (RFC1123) / mirror_url / target_disk 格式校验、必填字段检查 |

### DevOps / 部署

| 功能 | 状态 | 说明 |
|------|------|------|
| Docker Compose 一键部署 | [x] | 开发环境 (`docker-compose.yml`) + 生产环境 (`docker-compose.prod.yml`) |
| CI/CD Pipeline | [x] | GitHub Actions: 安全扫描 → 构建测试 → Docker 镜像构建 |
| 安全扫描 | [x] | GolangCI-Lint + Gosec + govulncheck + Trivy 多层安全审查 |
| Schema 迁移 | [x] | 启动时自动执行 `internal/dbmigrate/migrations/*.sql`,dirty state 启动失败 |
| Helm Chart / K8s 部署 | [ ] | 暂不支持 |
| Terraform Provider | [ ] | 暂未开发 |

---

## 节点生命周期

节点状态有 6 个,转换由后端集中校验(`api/node_status.go`),
非法跃迁会被拒绝并返回 `ErrIllegalTransition`。

| 状态 | 含义 | 进入方式 |
|---|---|---|
| `discovered` | PXE 自动发现,未操作员审批 | 未知 MAC 触发 `/pxe/boot/:mac` 时自动创建 |
| `pending` | 已注册,未开始装机 | `CreateNode` 默认值;`installing` 失败可回退到;`discovered` 经操作员审批可进入 |
| `installing` | 正在 PXE 装机 | `RebuildNode` / 显式 create with `status=installing` |
| `active` | 在线 + 装机完成 | PostInstall 回调;heartbeat 把 `offline` 拉回 |
| `maintenance` | 运维主动锁定 | 操作员显式设置 |
| `error` | 装机失败或异常 | 装机超时自动转;操作员显式设置 |
| `offline` | 心跳超时 | 后台 `StaleNodeChecker` 把 `active` 标记 |

```
   create
   ──────►  pending  ───────────► installing ─────────► active
                                     │   ▲                │
                              timeout│   │ rebuild        │ stale
                                     ▼   │                ▼
                                   error                offline
                                     ▲
                                     │ (任意终态都可 rebuild,
                                     │  或操作员显式 maintenance)
```

**关键不变量**

- 进入 `installing` 自动写 `installing_started_at = now()`,离开任意目标状态时清除。
- `installing` 超过 `INSTALL_TIMEOUT_MINUTES`(默认 120 min)未收到 active 回调
  → 后台 checker(同一 ticker,5 min 触发一次)将节点转 `error` + 清时间戳。
- Heartbeat **不允许**把 `installing` / `maintenance` / `pending` 直接提升为 `active`
  —— 防止 agent 心跳与 install-timeout watcher 竞争,或解除运维锁定。
- 每次状态变更都生成审计日志条目:`<from> -> <to>: <reason>`。
- **`discovered` 节点不会被 PXE 自动装机** — BootScript 看到 `discovered` 状态返回测试菜单
  并显示 "awaiting operator approval";操作员在 UI 配置 OS 类型/加密选项之后,
  调用 `POST /nodes/:id/rebuild` 触发 `discovered → installing`,然后正常装机。
- `POST /nodes/:id/rebuild` 在状态机就位后**异步**触发 IPMI `power cycle`(信号量限流,默认并发 8),
  配置了 BMC 的节点真正零接触装机。操作员可附加 `?no_power_cycle=1`
  跳过(故障排查 / 已经在维护窗口内手动操作时使用);全局可通过
  `IPMI_AUTO_POWER_CYCLE=false` 关闭。结果落 audit log,
  HTTP 响应里 `power_cycle: scheduled|skipped:no_bmc|disabled|skipped:opted_out`。

---

## 架构概览

> **完整交互式架构图见 [`docs/architecture.html`](./docs/architecture.html)** ——
> 单文件 HTML,Excalidraw 风格手绘图,5 个 tab 分别覆盖组件拓扑 / PXE 装机时序 /
> 节点状态机 / 后台 watcher / 节点注册流。双击文件即可在浏览器打开,无需 build。

下面是 ASCII 速览(精度有限,详情请看上面的 HTML):

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
                    │                ┌────▼────┐                   │
                    │                │ Postgres│                   │
                    │                │  :5432  │                   │
                    │                └─────────┘                   │
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
    │       (后端为本次装机颁发一次性 token,
    │        embed 为 ?t=<token>;node 状态打 installing_started_at)
    ▼
下载 Kernel + Initrd ──► 启动安装程序
    │
    ▼
Preseed 无人值守安装 ──► GET /api/v1/pxe/preseed/{mac}?t=<token>
    │       (验证 token 但不消费;installer 可重试)
    ▼
自动安装 OS + LUKS 加密 + TPM2 注册
    │
    ▼
PostInstall 脚本下载 ──► GET /api/v1/pxe/postinstall/{mac}?t=<token>
    │       (验证 + 消费 token;绑 client IP)
    ▼
PUT /api/v1/internal/nodes/{id}/status ──► 走状态机 installing → active
                                          (清 installing_started_at)
```

> 装机超时:`installing_started_at` 超过 `INSTALL_TIMEOUT_MINUTES`(默认 120 min)
> 仍未收到 active 回调时,后台 checker 自动将节点标 `error`。

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
│   │   │   ├── auth.go              # 认证 (Login / Me)
│   │   │   ├── nodes.go             # 节点 CRUD + 密钥轮换
│   │   │   ├── node_status.go       # 节点状态机 + 合法性转换检查
│   │   │   ├── users.go             # 用户管理 CRUD
│   │   │   ├── pxe.go               # PXE 脚本生成 (init/boot/preseed/postinstall)
│   │   │   ├── pxe_token.go         # 一次性装机 token 颁发/校验/消费
│   │   │   ├── dhcp.go              # DHCP 配置 & 保留管理
│   │   │   ├── dnsmasq.go           # dnsmasq 配置文件生成(同步执行函数)
│   │   │   ├── dnsmasq_scheduler.go # dnsmasq 重生成去抖调度器
│   │   │   ├── asset.go             # 启动资产上传 / 管理
│   │   │   ├── dashboard.go         # 仪表盘数据聚合
│   │   │   ├── audit.go             # 审计日志(同步写入 + 删除快照)
│   │   │   ├── ipmi.go              # IPMI 电源管理
│   │   │   ├── crypto.go            # AES-256-GCM 字段加密 (EncryptField/DecryptField)
│   │   │   ├── groups.go            # 节点分组 & 标签管理
│   │   │   ├── bulk.go              # 批量操作 (重建/删除/分组)
│   │   │   ├── heartbeat.go         # 节点心跳 + 离线/装机超时双 checker
│   │   │   ├── apikeys.go           # API Key 管理 (CRUD + 轮换)
│   │   │   ├── ratelimit.go         # 速率限制中间件
│   │   │   ├── websocket.go         # WebSocket SSH 代理
│   │   │   ├── utils.go             # JWT 生成/验证 + RBAC 中间件
│   │   │   └── notifications.go     # 通知管理
│   │   ├── config/             # 配置加载 (YAML + 环境变量)
│   │   ├── dbmigrate/          # golang-migrate 嵌入式 schema 迁移
│   │   │   └── migrations/     # 编号 SQL 文件 (000001_baseline 等)
│   │   ├── model/              # 数据模型 (GORM,仅 DTO,不驱动 schema)
│   │   └── sysutil/            # 系统工具 (网络接口探测等)
│   └── docs/                   # Swagger 自动生成文档
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
| **Schema 迁移** | golang-migrate/migrate(嵌入式 SQL 文件,启动时自动应用) |
| **密钥管理** | 字段级 AES-256-GCM 加密存储 (FIELD_ENCRYPTION_KEY) |
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
| **内部** | PUT | `/api/v1/internal/nodes/:id/status` | 节点状态回调(走状态机,仅限私网 IP / X-Internal-Token) |
| | POST | `/api/v1/internal/nodes/:id/heartbeat` | 节点心跳上报(仅 active/offline 状态允许提升为 active) |
| **健康** | GET | `/health/live` | 进程存活探针(永远 200) |
| | GET | `/health/ready` | 就绪探针(检查 DB 连通) |
| **系统** | GET | `/api/v1/system/interfaces` | 网络接口列表 |

---

## 配置

主要配置文件为 `config.yaml`，也支持环境变量覆盖：

| 环境变量 | 说明 | 默认值 |
|----------|------|--------|
| `DATABASE_URL` | PostgreSQL 连接串 | `postgresql://osbaka:password@localhost:5432/osbaka` |
| `PORT` | 后端监听端口 | `8000` |
| `SECRET_KEY` | JWT 签名密钥(release 模式下 < 32 字符或保留默认值会启动失败) | `change-this-in-production` |
| `GIN_MODE` | Gin 运行模式 | `debug` |
| `ADMIN_PASSWORD` | 管理员初始密码 | `admin` |
| `EXTERNAL_IP` | PXE 服务对外 IP | 自动检测 |
| `VITE_API_URL` | 前端 API 地址 | `http://localhost:8000/api/v1` |
| `VITE_GEMINI_API_KEY` | Gemini AI API Key (可选) | - |
| `FIELD_ENCRYPTION_KEY` | 字段级 AES-256-GCM 加密密钥 (hex, 64 字符) | - |
| `PXE_ASSETS_DIR` | 启动资产存储目录 | `/tftpboot` |
| `PXE_MIRROR_URL` | 全局 OS 镜像源 | 各发行版默认源 |
| `PXE_UBUNTU_MIRROR_URL` | Ubuntu 专用镜像源(优先级高于全局) | - |
| `PXE_DEBIAN_MIRROR_URL` | Debian 专用镜像源(优先级高于全局) | - |
| `PXE_DEBIAN_SECURITY_MIRROR_URL` | Debian security 仓镜像;留空走 `security.debian.org` 默认 | - |
| `PXE_UBUNTU_SECURITY_MIRROR_URL` | Ubuntu security 仓镜像;留空走 `security.ubuntu.com` 默认 | - |
| `PXE_REQUIRE_TOKEN` | 是否启用一次性 PXE token 校验,设 `false/0/no` 关闭(不建议) | `true` |
| `PXE_TOKEN_TTL_MINUTES` | PXE token 有效期;**未设时默认 = 2 × INSTALL_TIMEOUT_MINUTES** | (derived) |
| `INSTALL_TIMEOUT_MINUTES` | 装机超时阈值,超出后 `installing` 节点自动转 `error` | `120` |
| `IPMI_AUTO_POWER_CYCLE` | Rebuild 时是否自动通过 IPMI 重启目标节点,设 `false/0/no` 关闭 | `true` |
| `IPMI_POWER_CYCLE_CONCURRENCY` | 后端同时发起的 IPMI power-cycle 数量上限(信号量) | `8` |
| `PXE_AUTO_DISCOVER` | 未知 MAC PXE 触发时是否自动注册为 `discovered`,设 `false/0/no` 关闭(适用于共享 L2 场景) | `true` |

---

## 内网 / 离线部署

在没有公网出口的环境(企业内网、机房隔离段)装机时,以下几条**必须**配置到位,否则装机会卡在拉包阶段:

### 1. 内网 mirror 必须包含的内容

OS-Baka 假定 mirror 有标准 Debian 仓库布局:

```
http://mirror.intra/debian/
├── dists/<release>/main/                              # main archive (preseed 拉的就是这里)
└── dists/<release>/main/installer-amd64/current/      # netboot kernel/initrd
    └── images/netboot/debian-installer/amd64/{linux,initrd.gz}

http://mirror.intra/debian-security/
└── dists/<release>-security/main/                     # 必须独立!别跟 main 混在一起
```

**security 仓必须独立**:Debian 公网用 `security.debian.org` 域名,内网通常约定俗成放在 `<host>/debian-security/`。OS-Baka 默认会去 `security.debian.org`,**这在内网是死路** — 必须通过下面任一方式覆盖。

`postinstall.sh` 在装完后还会 `apt-get install tpm2-tools cryptsetup cryptsetup-initramfs`(仅在 LUKS + TPM 节点启用时),**这些包也必须在内网 mirror 里**;否则 TPM 自动解锁绑不上,机器每次开机要手动输 LUKS 密码。`/var/log/osbaka-tpm-setup.log` 会留 apt-get 的失败日志,运维 SSH 上去能查。

### 2. Mirror 覆盖入口

| 入口 | 适用场景 | 优先级 |
|---|---|---|
| `node.mirror_url` + `node.security_mirror_url` | 单节点特殊路由(如灾备节点指备份 mirror) | 最高 |
| `PXE_DEBIAN_MIRROR_URL` + `PXE_DEBIAN_SECURITY_MIRROR_URL` env | 部署级默认 | 中 |
| `DHCPConfig.mirror_url` + `DHCPConfig.security_mirror_url` | UI 配的 fleet-wide 默认 | 低 |
| 内置默认 | 公网默认 — 内网下错的就是它 | 兜底 |

典型内网 docker-compose 部署:

```yaml
backend:
  environment:
    PXE_DEBIAN_MIRROR_URL: http://mirror.intra/debian
    PXE_DEBIAN_SECURITY_MIRROR_URL: http://mirror.intra/debian-security
    EXTERNAL_IP: 10.0.0.10                # backend 在内网的 IP,PXE 客户端拉 preseed/postinstall 用
```

### 3. 其他内网常见坑

- **EXTERNAL_IP 必填**:不设的话 PXE iPXE 脚本会走 `request.Host` / 系统接口探测,常被路由到错误的 IP。dnsmasq 配置兜底是硬编码 `192.168.10.1`(`dnsmasq.go`)。
- **DNS 在 d-i 中可能失效**:postinstall 用 IP 访问 backend(不是域名),所以 EXTERNAL_IP 必须可路由。
- **NTP**:LUKS+TPM 节点对时钟敏感,若内网没有 NTP server 配置,装出来的机器系统时间漂移会触发 PCR 不匹配。建议 preseed 加 `kernel-command-line: time/zone=Asia/Shanghai` 之外,内网 mirror 旁边架 chrony。
- **boot kernel/initrd**:它们来自 `MirrorURL/dists/.../netboot/`,而**不是** `/tftpboot/`。常被以为 `/tftpboot/` 装好了内核就行 — 不是。

---

## 开发指南

### 后端开发

```bash
cd backend
go mod download

# 运行(启动时会同步执行 schema migrations)
go run ./cmd/server

# 测试
go test -v ./...

# 测试 + race detector(scheduler/heartbeat 等并发路径)
go test -race ./...

# Lint
golangci-lint run --timeout=5m

# 更新 Swagger 文档
swag init -g cmd/server/main.go
```

### Schema 迁移

Schema 由 `internal/dbmigrate/migrations/NNNNNN_name.{up,down}.sql` 驱动,
通过 `embed.FS` 打包进 server 二进制,启动时自动 apply。
GORM model **仅作为 query-time DTO**,**不再驱动 schema**——加字段必须同时写 migration。

```bash
# 安装 migrate CLI(本地调试用,server 自带执行)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

# 应用全部待办迁移
migrate -path backend/internal/dbmigrate/migrations \
        -database "$DATABASE_URL" up

# 单步回滚
migrate -path backend/internal/dbmigrate/migrations \
        -database "$DATABASE_URL" down 1

# 新建迁移:依次取下一个编号
# vi backend/internal/dbmigrate/migrations/000NNN_my_change.{up,down}.sql
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
