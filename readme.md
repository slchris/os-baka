# OS-Baka

一个现代化的 PXE 启动和资产管理系统，提供安全的操作系统部署、加密管理和节点监控功能。

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

## 项目结构

```
os-baka/
├── frontend/         # React + TypeScript + Vite 前端
├── backend/          # Go (Gin + GORM + PostgreSQL) 后端
│   ├── cmd/server/   # 应用入口
│   ├── internal/
│   │   ├── api/      # HTTP Handler 层
│   │   ├── config/   # 配置加载
│   │   ├── model/    # 数据模型 & 数据库初始化
│   │   └── sysutil/  # 系统工具
│   └── docs/         # Swagger 自动生成文档
├── docker/           # Docker Compose 配置
├── pxe-services/     # PXE 服务 (dnsmasq + TFTP + iPXE)
├── scripts/          # 部署脚本
└── config.yaml       # 统一配置文件
```

## 技术栈

- **前端**: React 19 + TypeScript + Vite + Tailwind CSS v4 + React Router v7
- **后端**: Go 1.24 + Gin + GORM + PostgreSQL
- **PXE**: dnsmasq (DHCP/TFTP) + iPXE chainloading
- **部署**: Docker + Docker Compose

## 默认登录

- **用户名**: `admin`
- **密码**: `admin`

> **注意**：请在生产环境通过 `ADMIN_PASSWORD` 环境变量设置管理员密码。

## 主要功能

- 节点/资产管理
- PXE 启动服务配置（BIOS + UEFI）
- DHCP 配置管理 & 静态地址保留
- 加密管理（LUKS + TPM2）
- 自动化系统安装（Ubuntu/Debian Preseed, CentOS/RHEL Kickstart）
- 实时仪表盘监控
- 暗色/亮色主题切换
- Swagger API 文档

## API 文档

启动后端后，访问 Swagger UI：
- http://localhost:8000/api/v1/docs/index.html

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
| `VAULT_ENABLED` | 启用 Vault 密钥管理 | `false` |
| `VAULT_ADDR` | Vault 服务地址 | `http://vault:8200` |
| `VAULT_TOKEN` | Vault 认证 Token | - |
