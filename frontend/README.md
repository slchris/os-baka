# OS-Baka Frontend

React + TypeScript 前端应用，采用 Apple 风格设计。

## 技术栈

- **React 19** - UI 框架
- **TypeScript** - 类型安全
- **Vite** - 构建工具
- **React Router 7** - 路由管理
- **Axios** - HTTP 客户端
- **Lucide React** - 图标库

## 开发

```bash
# 安装依赖
npm install

# 启动开发服务器
npx vite --host 127.0.0.1 --port 3000

# 构建生产版本
npm run build

# 预览生产构建
npm run preview
```

## 环境配置

创建 `.env.local` 文件：

```env
VITE_API_URL=http://127.0.0.1:8000/api/v1
```

## 项目结构

```
frontend/
├── index.html           # HTML 入口
├── vite.config.ts       # Vite 配置
├── tsconfig.json        # TypeScript 配置
├── nginx.conf           # Nginx 部署配置
└── src/
    ├── index.tsx        # 应用入口
    ├── App.tsx          # 主应用布局
    ├── router.tsx       # 路由配置
    ├── types.ts         # TypeScript 类型定义
    ├── components/      # 可复用组件
    │   └── Sidebar.tsx
    ├── pages/           # 页面组件
    │   ├── Login.tsx
    │   ├── Dashboard.tsx
    │   ├── Nodes.tsx
    │   └── ...
    └── services/        # API 和业务逻辑
        ├── apiClient.ts
        └── backendService.ts
```

## 可用路由

- `/login` - 登录页
- `/dashboard` - 仪表板
- `/nodes` - 节点管理
- `/keyvault` - 密钥库
- `/webshell` - Web Shell
- `/notifications` - 通知
- `/documentation` - 文档
- `/settings` - 设置

## 特性

- ✅ 完整的路由系统（React Router）
- ✅ 受保护路由（需登录）
- ✅ 深色模式支持
- ✅ 响应式设计
- ✅ TypeScript 类型安全
- ✅ API 统一管理

