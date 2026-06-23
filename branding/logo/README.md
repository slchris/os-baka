# OS-Baka Logo — Rack Lock

OS-Baka 的品牌主标。一个硬直角服务器机箱轮廓,内部镂空形成顶部机架导轨 + 两个驱动仓,中央熔接一个锁孔 —— **机架(裸金属)** 与 **锁孔(全盘加密 / LUKS)** 一图双关。

极简几何 · 单色 · 单一 `<path>` + `fill-rule="evenodd"`(真负空间镂空,任意底色通用)。

## 文件

| 文件 | 用途 |
|------|------|
| `showcase.html` | 终稿展示页(缩放矩阵 / 底色 / Lockup / 应用预览 / 下载),浏览器打开 |
| `svg/osbaka-mark.svg` | **主标**。`currentColor`,≥32px 场景(侧栏头部 / 登录 / 文档) |
| `svg/osbaka-favicon.svg` | **Favicon 变体**。精简为 1 导轨 + 1 加大锁孔,≤24px 场景 |
| `png/osbaka-mark-{32..1024}.png` | 主标多尺寸位图(黑色填充) |
| `png/osbaka-mark-512-ondark.png` | 主标白色 / 深底 |
| `png/osbaka-favicon-{16..64}.png` | Favicon 变体小尺寸 |

## 用法

颜色由 CSS `color:` 或父级继承控制(`fill="currentColor"`)。

```html
<!-- 内联,跟随文字颜色 -->
<span style="color:#f2f3f5">
  <!-- 粘贴 svg/osbaka-mark.svg 内容 -->
</span>

<!-- favicon -->
<link rel="icon" type="image/svg+xml" href="/branding/logo/svg/osbaka-favicon.svg">
```

## 重新导出 PNG

需要 `rsvg-convert`(`brew install librsvg`)。`currentColor` 在独立栅格化时不解析,先替换成具体色:

```bash
sed 's/currentColor/#111111/' svg/osbaka-mark.svg > /tmp/m.svg
rsvg-convert -w 512 -h 512 /tmp/m.svg -o png/osbaka-mark-512.png
```

## 设计取舍

- **实心而非描边**:实心填充缩小时远比细平行描边稳。
- **硬角 (rx=6)**:刻意保留;大圆角会被读成盾牌 / 记事本,硬角才说"金属盒子"。
- **锁孔单一连通**:头部圆 + 同心内孔(evenodd 镂空)+ 熔接喇叭口槽,一个路径画完,避免"圆环 + 浮块"的断裂感。
