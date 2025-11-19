# OS Baka - Apple 风格设计系统

## 概述

本设计系统基于 Apple App Store 网站的视觉风格，提取并整理了核心设计元素，用于 OS Baka 项目的 UI 开发。

## 设计原则

### 1. 简洁优先
- 减少视觉噪音
- 突出核心内容
- 留白空间充足

### 2. 一致性
- 统一的组件样式
- 一致的交互模式
- 标准化的间距系统

### 3. 响应式
- 移动端优先设计
- 流畅的断点过渡
- 自适应布局

### 4. 性能
- 轻量级实现
- 优化的加载体验
- 平滑的动画效果

## 颜色系统

### 系统颜色（明亮模式）
```scss
// 主色
--systemPrimary: rgb(0, 0, 0);           // 黑色文本
--systemSecondary: rgb(128, 128, 128);   // 次要文本
--systemTertiary: rgb(170, 170, 170);    // 三级文本

// 背景色
--defaultBackground: rgb(255, 255, 255);  // 白色背景
--footerBg: rgb(246, 246, 246);          // 页脚背景

// 强调色
--keyColor: rgb(0, 113, 227);            // Apple 蓝色
```

### 系统颜色（深色模式）
```scss
@media (prefers-color-scheme: dark) {
  --systemPrimary: rgb(255, 255, 255);
  --systemSecondary: rgb(152, 152, 157);
  --systemTertiary: rgb(128, 128, 128);
  --defaultBackground: rgb(0, 0, 0);
  --footerBg: rgb(29, 29, 31);
}
```

### 强调色系统
```scss
// 卡片强调色示例
--accent-purple: rgb(158, 71, 162);
--accent-orange: rgb(240, 142, 24);
--accent-yellow: rgb(252, 198, 92);
--accent-blue: rgb(1, 87, 243);
--accent-green: rgb(120, 165, 63);
```

## 排版系统

### 字体家族
```scss
// SF Pro 字体（Apple 官方字体）
font-family: 'SF Pro', 'SF Pro Text', -apple-system, BlinkMacSystemFont, 
             'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
```

### 字体大小与权重

```scss
// 标题
--title-1: 600 28px/1.1 'SF Pro';        // 大标题
--title-2: 600 22px/1.2 'SF Pro';        // 中标题
--title-3: 600 20px/1.2 'SF Pro';        // 小标题

// 正文
--body: 400 17px/1.47 'SF Pro Text';          // 标准正文
--body-emphasized: 600 17px/1.47 'SF Pro Text'; // 强调正文

// 小字
--footnote: 400 13px/1.38 'SF Pro Text';          // 脚注
--footnote-emphasized: 600 13px/1.38 'SF Pro Text'; // 强调脚注
```

## 布局系统

### 断点
```scss
// 参考 App Store 的断点设置
$breakpoint-xsmall: 0px;      // 移动端
$breakpoint-small: 740px;     // 平板
$breakpoint-medium: 1000px;   // 小屏桌面
$breakpoint-large: 1320px;    // 大屏桌面
$breakpoint-xlarge: 1680px;   // 超大屏

// 侧边栏可见断点
@media (--sidebar-visible) {
  // min-width: 740px
}

@media (--sidebar-large-visible) {
  // min-width: 1000px
}
```

### 网格系统
```scss
// 容器最大宽度
--max-width-small: 740px;
--max-width-medium: 1000px;
--max-width-large: 1320px;
--max-width-xlarge: 1680px;

// 侧边栏宽度
$global-sidebar-width: 260px;
$global-sidebar-width-large: 280px;
```

### 间距系统
```scss
// 基础间距单位：8px
--spacing-xs: 4px;    // 0.5x
--spacing-sm: 8px;    // 1x
--spacing-md: 16px;   // 2x
--spacing-lg: 24px;   // 3x
--spacing-xl: 32px;   // 4x
--spacing-2xl: 48px;  // 6x
--spacing-3xl: 64px;  // 8x
```

## 组件样式

### 1. 导航栏 (Navigation)

#### 结构
```html
<nav class="navigation">
  <div class="navigation__header">
    <div class="logo-container">
      <!-- Logo + Platform Selector -->
    </div>
    <div class="search-container">
      <!-- Search Input -->
    </div>
  </div>
  <div class="navigation__content">
    <ul class="navigation-items">
      <!-- Navigation Items -->
    </ul>
  </div>
</nav>
```

#### 样式特点
- 固定高度：44px (移动端) / 100vh (桌面端)
- Sticky 定位
- 平滑的悬停效果
- 圆角图标：18-32px

### 2. 卡片 (Cards)

#### Today Card 样式
```scss
.today-card {
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  transition: transform 0.3s ease, box-shadow 0.3s ease;
  
  &:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  }
}
```

#### 卡片比例
- 纵向 (Portrait): 0.75 (3:4)
- 横向 (Landscape): 1.78 (16:9)

### 3. 应用图标 (App Icons)

#### 圆角处理
```scss
.app-icon {
  border-radius: 22.5%; // iOS 风格圆角
  overflow: hidden;
  
  // 尺寸变体
  &--small { width: 48px; height: 48px; }
  &--medium { width: 64px; height: 64px; }
  &--large { width: 128px; height: 128px; }
}
```

### 4. 按钮 (Buttons)

#### 主按钮
```scss
.button-primary {
  background: var(--keyColor);
  color: white;
  border: none;
  border-radius: 8px;
  padding: 12px 24px;
  font: var(--body-emphasized);
  cursor: pointer;
  transition: opacity 0.2s ease;
  
  &:hover {
    opacity: 0.85;
  }
  
  &:active {
    opacity: 0.7;
  }
}
```

#### 次要按钮
```scss
.button-secondary {
  background: transparent;
  color: var(--keyColor);
  border: 1px solid rgba(0, 113, 227, 0.3);
  border-radius: 8px;
  padding: 12px 24px;
  font: var(--body);
  
  &:hover {
    background: rgba(0, 113, 227, 0.1);
  }
}
```

### 5. 输入框 (Inputs)

#### 搜索框
```scss
.search-input {
  background: rgba(120, 120, 128, 0.12);
  border: none;
  border-radius: 10px;
  padding: 8px 16px 8px 36px; // 左侧留给图标
  font: var(--body);
  color: var(--systemPrimary);
  
  &::placeholder {
    color: var(--systemTertiary);
  }
  
  &:focus {
    outline: none;
    background: rgba(120, 120, 128, 0.16);
  }
}
```

#### 文本输入框
```scss
.text-input {
  border: 1px solid rgba(0, 0, 0, 0.12);
  border-radius: 8px;
  padding: 12px 16px;
  font: var(--body);
  transition: border-color 0.2s ease;
  
  &:focus {
    outline: none;
    border-color: var(--keyColor);
  }
}
```

### 6. 表格 (Tables)

```scss
.table {
  width: 100%;
  border-collapse: collapse;
  
  th {
    font: var(--body-emphasized);
    color: var(--systemSecondary);
    text-align: left;
    padding: 12px 16px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.1);
  }
  
  td {
    font: var(--body);
    padding: 16px;
    border-bottom: 1px solid rgba(0, 0, 0, 0.05);
  }
  
  tr:hover {
    background: rgba(0, 0, 0, 0.02);
  }
}
```

## 动画与过渡

### 标准过渡
```scss
// 快速过渡（按钮、链接）
--transition-fast: 150ms cubic-bezier(0.4, 0, 0.2, 1);

// 标准过渡（卡片、模态框）
--transition-standard: 300ms cubic-bezier(0.4, 0, 0.2, 1);

// 慢速过渡（页面切换）
--transition-slow: 500ms cubic-bezier(0.4, 0, 0.2, 1);
```

### Easing 函数
```scss
// App Store 使用的 easing
--ease-out: cubic-bezier(0, 0, 0.2, 1);    // 减速
--ease-in: cubic-bezier(0.4, 0, 1, 1);     // 加速
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1); // 平滑
```

### 动画示例

#### 淡入淡出
```scss
@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes fadeOut {
  from { opacity: 1; }
  to { opacity: 0; }
}
```

#### 滑动进入
```scss
@keyframes slideIn {
  from {
    transform: translateY(20px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}
```

## 图标系统

### SF Symbols
Apple 使用 SF Symbols 图标系统，建议使用：
- 线性图标为主
- 2px 描边粗细
- 填充与线性结合使用

### 常用图标尺寸
```scss
--icon-xs: 16px;
--icon-sm: 20px;
--icon-md: 24px;
--icon-lg: 32px;
--icon-xl: 48px;
```

## 阴影系统

```scss
// 卡片阴影
--shadow-card: 0 2px 8px rgba(0, 0, 0, 0.1);
--shadow-card-hover: 0 8px 24px rgba(0, 0, 0, 0.15);

// 模态框阴影
--shadow-modal: 0 12px 48px rgba(0, 0, 0, 0.2);

// 下拉菜单阴影
--shadow-dropdown: 0 4px 16px rgba(0, 0, 0, 0.12);
```

## 可访问性

### 焦点状态
```scss
// 键盘焦点样式
.focus-visible:focus-visible {
  outline: 2px solid var(--keyColor);
  outline-offset: 2px;
}
```

### 对比度
- 文本对比度 ≥ 4.5:1（WCAG AA）
- 大文本对比度 ≥ 3:1

### 语义化标签
- 使用正确的 HTML5 语义标签
- ARIA 属性增强可访问性
- 键盘导航支持

## 代码示例

### Svelte 组件模板
```svelte
<script lang="ts">
  export let title: string;
  export let subtitle: string = '';
</script>

<div class="card">
  <h2 class="card__title">{title}</h2>
  {#if subtitle}
    <p class="card__subtitle">{subtitle}</p>
  {/if}
  <slot />
</div>

<style lang="scss">
  .card {
    background: var(--defaultBackground);
    border-radius: 12px;
    padding: var(--spacing-lg);
    box-shadow: var(--shadow-card);
    transition: var(--transition-standard);
    
    &:hover {
      box-shadow: var(--shadow-card-hover);
      transform: translateY(-2px);
    }
  }
  
  .card__title {
    font: var(--title-2);
    color: var(--systemPrimary);
    margin: 0 0 var(--spacing-sm);
  }
  
  .card__subtitle {
    font: var(--body);
    color: var(--systemSecondary);
    margin: 0 0 var(--spacing-md);
  }
</style>
```

## 实施建议

### 1. 组件库开发顺序
1. 基础组件（按钮、输入框、卡片）
2. 导航组件（导航栏、侧边栏、面包屑）
3. 数据展示（表格、列表、徽章）
4. 反馈组件（模态框、通知、加载）

### 2. CSS 架构
- 使用 CSS Variables 实现主题
- SCSS 用于组件样式
- BEM 命名规范
- 响应式优先

### 3. 性能优化
- 懒加载图片
- 代码分割
- 树摇优化
- 压缩资源

### 4. 浏览器兼容性
- 支持现代浏览器（Chrome、Firefox、Safari、Edge）
- 优雅降级处理
- CSS Grid/Flexbox 布局

## 参考资源

- [Apple Human Interface Guidelines](https://developer.apple.com/design/human-interface-guidelines/)
- [SF Symbols](https://developer.apple.com/sf-symbols/)
