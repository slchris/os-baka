# 前端问题排查指南

## 已修复的问题

### ✅ Vite 7 与 Svelte 4 兼容性问题

**问题描述：**
```
Internal server error: Unrecognized option 'hmr'
Plugin: vite-plugin-svelte:compile
File: /Users/chris/git/vc/os-baka/frontend/src/App.svelte
```

**原因分析：**
Vite 7.x 版本与 `@sveltejs/vite-plugin-svelte` 6.x 存在兼容性问题。Svelte 编译器在新版本中移除了 `hmr` 选项，导致编译失败。

**解决方案：**
降级到经过验证的稳定版本组合：
- Vite: `5.0.11` (替代 7.2.2)
- @sveltejs/vite-plugin-svelte: `3.0.1` (替代 6.2.1)
- Svelte: `4.2.8` (保持不变)

```json
// package.json - 已修复
{
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^3.0.1",  // ✅ 降级
    "svelte": "^4.2.8",  // ✅ 保持
    "vite": "^5.0.11"  // ✅ 降级
  }
}
```

**重新安装依赖：**
```bash
cd frontend
rm -rf node_modules package-lock.json
npm install
npm run dev
```

---

### ✅ Sass 导入路径错误

**问题描述：**
```
[plugin:vite-plugin-svelte:preprocess] Error while preprocessing
[sass] Can't find stylesheet to import.
@import "./src/styles/variables.scss";
```

**原因分析：**
在 `vite.config.ts` 中使用 `css.preprocessorOptions.scss.additionalData` 自动导入样式文件时，路径解析出现问题。这个配置会在每个 SCSS 文件编译前注入代码，但路径是相对于工作目录而非文件本身。

**解决方案：**
移除 `vite.config.ts` 中的 `additionalData` 配置，改为在 `global.scss` 中显式导入：

```typescript
// vite.config.ts - 已修复
export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8000',
        changeOrigin: true,
      }
    }
  }
  // ❌ 移除了这部分
  // css: {
  //   preprocessorOptions: {
  //     scss: {
  //       additionalData: `@import "./src/styles/variables.scss";`
  //     }
  //   }
  // }
})
```

```scss
// src/styles/global.scss
@import './variables.scss';  // ✅ 在这里导入

// ... 其他全局样式
```

---

## 常见问题

### npm install 失败

**问题：** 依赖冲突或网络问题

**解决方案：**
```bash
# 清除缓存
rm -rf node_modules package-lock.json
npm cache clean --force

# 重新安装
npm install

# 或使用 --legacy-peer-deps
npm install --legacy-peer-deps
```

### 端口 5173 被占用

**问题：** `Port 5173 is already in use`

**解决方案：**
```bash
# 查找进程
lsof -i :5173

# 杀死进程
kill -9 <PID>

# 或修改 vite.config.ts 中的端口
server: {
  port: 5174  // 使用其他端口
}
```

### Svelte 组件报错

**问题：** 组件导入路径错误

**解决方案：**
确保使用相对路径导入：
```typescript
// ✅ 正确
import Button from './components/Button.svelte'
import { api } from './lib/api'

// ❌ 错误
import Button from 'components/Button.svelte'
```

### 类型错误

**问题：** TypeScript 类型检查失败

**解决方案：**
```bash
# 运行类型检查
npm run check

# 查看具体错误
npx svelte-check
```

### API 请求失败

**问题：** CORS 或代理配置问题

**解决方案：**
1. 确认后端正在运行（http://localhost:8000）
2. 检查 `vite.config.ts` 中的代理配置
3. 确认后端 CORS 配置正确

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8000',
      changeOrigin: true,
      rewrite: (path) => path  // 可选：路径重写
    }
  }
}
```

### 样式不生效

**问题：** CSS 变量未定义或被覆盖

**解决方案：**
1. 确认 `global.scss` 已在 `main.ts` 中导入
2. 检查浏览器开发者工具中的 CSS 变量值
3. 确认组件使用的是正确的 CSS 变量名

```scss
// 查看所有可用的 CSS 变量
// src/styles/variables.scss

// 在组件中使用
.my-element {
  color: var(--systemPrimary);
  background: var(--defaultBackground);
  padding: var(--spacing-md);
}
```

### npm 安全警告

**问题：** `5 moderate severity vulnerabilities`

**解决方案：**
```bash
# 查看详情
npm audit

# 自动修复（谨慎使用）
npm audit fix

# 强制修复（可能破坏兼容性）
npm audit fix --force
```

对于开发依赖的警告，通常可以安全忽略。生产部署前再处理。

---

## 开发提示

### 热重载
- Vite 支持 HMR（Hot Module Replacement）
- 保存文件后自动更新浏览器
- 如果热重载失败，刷新浏览器即可

### Svelte 开发者工具
安装浏览器扩展：
- Chrome: [Svelte Devtools](https://chrome.google.com/webstore/detail/svelte-devtools/ckolcbmkjpjmangdbmnkpjigpkddpogn)
- Firefox: [Svelte Devtools](https://addons.mozilla.org/en-US/firefox/addon/svelte-devtools/)

### 调试技巧
```svelte
<script>
  // 使用 $: 响应式语句调试
  $: console.log('当前状态:', someVariable)
  
  // 在模板中输出调试信息
  {@debug someVariable}
</script>
```

### 性能优化
```svelte
<script>
  import { onMount, onDestroy } from 'svelte'
  
  let cleanup
  
  // 组件挂载后执行
  onMount(() => {
    console.log('组件已挂载')
    
    // 返回清理函数
    return () => {
      console.log('组件即将卸载')
    }
  })
</script>
```

### 推荐的 VSCode 扩展
- **Svelte for VS Code** - 官方 Svelte 支持
- **Svelte Intellisense** - 智能提示
- **Prettier - Code formatter** - 代码格式化
- **ESLint** - 代码检查

---

## 版本兼容性参考

### 当前稳定版本组合 ✅
```json
{
  "devDependencies": {
    "@sveltejs/vite-plugin-svelte": "^3.0.1",
    "svelte": "^4.2.8",
    "vite": "^5.0.11",
    "typescript": "^5.0.2",
    "sass": "^1.69.5"
  }
}
```

### 升级路径（未来）
如果需要升级到 Svelte 5 或 Vite 6+：
1. 先阅读官方迁移指南
2. 在分支中测试
3. 逐个更新依赖
4. 运行完整测试

---

## 需要帮助？

- 查看 [Vite 文档](https://vitejs.dev/)
- 查看 [Svelte 文档](https://svelte.dev/)
- 查看 [项目快速开始指南](../GETTING-STARTED.md)
- 查看 [设计系统文档](../docs/design-system.md)
- 查看 [项目状态文档](../docs/project-status.md)

---

**最后更新：** 2025-11-17
