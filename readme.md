# OS Baka

<div align="center">

🚀 **PXE 服务与物理资产管理系统**

具有 Apple 风格 UI 的现代化 PXE 网络启动服务和物理资产管理平台

[文档](./docs/) · [开发计划](./docs/development-plan.md) · [设计系统](./docs/design-system.md)

</div>

---

## ✨ 特性

- 🎨 **Apple 风格 UI** - 基于 Apple App Store 设计系统
- 📦 **资产管理** - IP、MAC、资产标签、USB Key 统一管理
- 🖥️ **PXE 服务** - 支持 macOS、Linux、Windows 网络启动
- 🔒 **自动加密** - FileVault、BitLocker、LUKS 自动配置
- 🚀 **节点管理** - 远程重建、电源管理、批量操作
- 🐳 **容器化部署** - 支持 Docker 和 Kubernetes
- 💻 **Web Shell** - 基于 WebSocket 的远程终端
- 🔑 **SSH 密钥管理** - 密钥生成、分发、轮换

## 🏗️ 技术栈

### 后端
- **FastAPI** - 现代化 Python Web 框架
- **PostgreSQL** - 关系型数据库
- **SQLAlchemy** - ORM
- **Celery** - 异步任务队列
- **Redis** - 缓存和消息队列

### 前端
- **Svelte** - 轻量级响应式框架
- **TypeScript** - 类型安全
- **Vite** - 快速构建工具
- **SCSS** - CSS 预处理器
- **Axios** - HTTP 客户端

### 基础设施
- **Docker** - 容器化
- **Nginx** - 反向代理
- **dnsmasq** - DHCP + TFTP 服务

## 🚀 快速开始

### 使用 Docker Compose（推荐）

```bash
# 克隆项目
git clone https://github.com/yourusername/os-baka.git
cd os-baka

# 启动所有服务
cd docker
docker-compose up -d

# 查看日志
docker-compose logs -f
```

访问：
- 前端: http://localhost:5173
- 后端 API: http://localhost:8000
- API 文档: http://localhost:8000/api/docs

### 本地开发

#### 1. 后端

```bash
cd backend

# 创建虚拟环境
python3 -m venv venv
source venv/bin/activate  # Windows: venv\Scripts\activate

# 安装依赖
pip install -r requirements.txt

# 配置环境变量
cp .env.example .env
# 编辑 .env 文件配置数据库等信息

# 运行数据库迁移
alembic upgrade head

# 启动开发服务器
uvicorn app.main:app --reload
```

#### 2. 前端

```bash
cd frontend

# 安装依赖
npm install

# 启动开发服务器
npm run dev
```

#### 3. 数据库

```bash
# 使用 Docker 启动 PostgreSQL
docker run -d \
  --name osbaka-db \
  -e POSTGRES_DB=osbaka \
  -e POSTGRES_USER=osbaka \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15-alpine
```

## 📚 文档

- [开发计划](./docs/development-plan.md)
- [设计系统](./docs/design-system.md)
- [API 文档](http://localhost:8000/api/docs) (运行后访问)

## 🗂️ 项目结构

```
os-baka/
├── backend/              # FastAPI 后端
│   ├── app/
│   │   ├── api/         # API 路由
│   │   ├── core/        # 核心配置
│   │   ├── models/      # 数据模型
│   │   └── schemas/     # Pydantic Schema
│   ├── alembic/         # 数据库迁移
│   └── requirements.txt
├── frontend/            # Svelte 前端
│   ├── src/
│   │   ├── components/  # UI 组件
│   │   ├── pages/       # 页面
│   │   ├── lib/         # 工具库
│   │   └── styles/      # 样式文件
│   └── package.json
├── docker/              # Docker 配置
│   └── docker-compose.yml
├── pxe-services/        # PXE 服务配置
├── k8s/                 # Kubernetes 配置
└── docs/                # 文档
```

## 🛠️ 开发进度

- [x] 项目初始化
- [x] 后端基础框架
- [x] 前端基础框架
- [x] 资产管理 API
- [x) 资产管理 UI
- [x] 用户认证系统
- [ ] PXE 服务实现
- [ ] 加密安装功能
- [ ] 节点管理功能
- [ ] Web Shell
- [ ] SSH 密钥管理
- [ ] Kubernetes 支持

## 📝 API 示例

### 创建资产

```bash
curl -X POST http://localhost:8000/api/v1/assets \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "macbook-pro-01",
    "ip_address": "192.168.1.100",
    "mac_address": "00:11:22:33:44:55",
    "asset_tag": "AST-001",
    "os_type": "macos",
    "encryption_enabled": true
  }'
```

### 列出所有资产

```bash
curl http://localhost:8000/api/v1/assets
```

## 🙏 致谢

- UI 设计灵感来自 [Apple App Store](https://apps.apple.com)
- 图标来自 [SF Symbols](https://developer.apple.com/sf-symbols/)

---

<div align="center">
Made with ❤️ by the OS Baka Team
</div>