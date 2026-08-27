# 可复用前端提示词:React + Vite + Tailwind 4 + shadcn/ui 「奶油绿」Dashboard

> 用途:从 0 启动任何 Go 系项目(Go 后端 + Web 管理界面)时,把下面整段提示词交给 AI,即可生成与本项目(octopus)同技术栈、同美术风格的前端。
> 来源:由 `octopus` 项目前端反编译归纳,未经删减可用的完整模板。

---

## 【提示词开始】

创建一个 Go 项目的 Web 管理界面,要求如下。

## 一、技术栈(必须一致)

- 框架:React 19 + TypeScript + Vite(构建)。允许 React Compiler(`@vitejs/plugin-react` + `babel-plugin-react-compiler`)。
- 包管理:`pnpm`。脚手架:手写 `vite.config.ts`,不使用 create-vite 的默认模板。
- 样式:Tailwind CSS 4(通过 `@tailwindcss/postcss` + `postcss.config.mjs`),入口 `src/globals.css`,使用 `tw-animate-css`。
- UI 组件:shadcn/ui(`components.json`,style=`new-york`,iconLibrary=`lucide`,baseColor=`neutral`),底层 Radix UI。
- 状态管理:Zustand(按需用于跨模块共享状态)。
- 数据请求与缓存:TanStack Query(recommend 用于所有 API 调用,统一 mutation/query hooks)。
- 国际化:`use-intl`,内置三语 `locales/{en,zh_hans,zh_hant}.json`,默认中文。
- 图表:`recharts`;动效:`motion`(framer-motion 系);虚拟列表 `@tanstack/react-virtual`;拖拽 `@hello-pangea/dnd`;Toast `sonner`;日期 `dayjs`/`date-fns`;图标 `lucide-react`;动画数字、JSON 查看器按需。
- 目录别名 `@` → `src`。

## 二、工程结构(约定一致)

```
web/
  index.html          # 含内联主题初始化 script + SVG 加载动画
  vite.config.ts      # base:'./', build.outDir => ../static/out, dev proxy /api => 127.0.0.1:8080
  components.json     # shadcn 配置(new-york)
  postcss.config.mjs
  tsconfig.json
  src/
    main.tsx          # ThemeProvider / QueryClientProvider / LocaleProvider / TooltipProvider / Toaster
    globals.css       # Tailwind + 全部 CSS 变量(otklch) + 背景光斑
    app.tsx           # 路由或页面分发容器
    components/
      app-shell.tsx     # 全局布局:Dock 导航 + 顶栏 + 内容区
      common/           # AnimatedNumber, CopyButton, PageActions, VirtualizedGrid 等通用组件
      modules/<feature>/  # 功能模块:index.tsx + Card.tsx + Form.tsx + Create.tsx(编辑)等
      ui/               # shadcn/ui 生成的组件
    api/                # client.ts(axios/fetch 封装)+ queries.ts(所有 useQuery/useMutation)+ 按资源分文件
    stores/             # zustand store
    hooks/  lib/  provider/  locales/
  components.json
```

## 三、美术风格(严格复刻「奶油绿」风格)

### 3.1 设计语言

- 品牌色为「柔和绿」,佐以米黄/奶油中性色。整体是低对比、温柔、治愈的 dashboard 观感;大面积圆角卡片 + 细浅边框 + 微弱阴影。
- 全局圆角 `--radius: 0.5rem`,但卡片类组件使用 `rounded-3xl`(内容卡)、`rounded-2xl`(内嵌指标块)、`rounded-xl`(小元素)。
- 支持深浅双主题,由 `html.dark` 类切换(`@custom-variant dark`),首次进入跟随系统,可手动切换并持久化。

### 3.2 主题色板(直接复制 oklch 值到 :root 与 .dark)

Light:
```css
:root {
  --background: oklch(0.9331 0.0081 98.8844);        /* 奶油米背景 */
  --foreground: oklch(0.3514 0.0250 84.4589);        /* 深灰褐文字 */
  --card: oklch(0.9891 0.0017 325.5900);             /* 近白卡片 */
  --card-foreground: oklch(0.3514 0.0250 84.4589);
  --popover: oklch(0.9796 0.0057 84.5661);
  --primary: oklch(0.6205 0.1199 144.8607);          /* 品牌绿 */
  --primary-foreground: oklch(0.9789 0.0101 145.4977);
  --secondary: oklch(0.9204 0.0149 98.2970);
  --muted: oklch(0.9399 0.0124 91.5213);
  --muted-foreground: oklch(0.5192 0.0198 84.5869);
  --accent: oklch(0.6800 0.1103 148.0832);           /* 亮绿 */
  --accent-foreground: oklch(0.9792 0.0091 150.6929);
  --destructive: oklch(0.5787 0.1500 34.8321);
  --border: oklch(0.8795 0.0142 88.6926);            /* 浅米边框 */
  --input: oklch(0.8795 0.0142 88.6926);
  --ring: oklch(0.6205 0.1199 144.8607);
  --chart-1..5: oklch(0.68→0.44 / 0.12→0.08 / 145°—155°); /* 绿色系渐变序列 */
  --sidebar: oklch(0.9891 0.0017 325.5900);
  --radius: 0.5rem;
  --shadow-*: 低对比柔和阴影(基准 0 1px 3px 0px hsl(0 0% 0% / 0.10));
}
```
Dark:
```css
.dark {
  --background: oklch(0.2809 0.0264 153.4040);        /* 深灰绿背景 */
  --card: oklch(0.3196 0.0217 150.3385);
  --primary: oklch(0.6990 0.1292 144.8727);
  --foreground: oklch(0.9200 0.0104 81.7947);
  --muted-foreground: oklch(0.6801 0.0151 88.7221);
  --sidebar: oklch(0.2593 0.0275 159.0885);
  --border: oklch(0.4006 0.0210 145.1704);
}
```

### 3.3 背景氛围光斑(必做)

`body::before` / `body::after` 两个绝对定位的柔光椭圆:
```css
body::before { width: 300px; height: 300px; background: radial-gradient(circle at 30% 20%, oklch(0.68 0.05 150) 0%, transparent 70%); top: 10%; right: 10%; filter: blur(40px); opacity: 0.6; z-index: -1; }
body::after  { width: 250px; height: 250px; background: radial-gradient(circle at 60% 40%, oklch(0.82 0.03 200) 0%, transparent 70%); bottom: 10%; left: 5%; filter: blur(40px); opacity: 0.6; }
```

### 3.4 布局与组件范式

- **AppShell 布局**:桌面端 `max-w-6xl` 居中,左侧 Dock 导航(`md:sticky` 垂直图标栏,`rounded-3xl` 悬浮容器)+ 顶栏(大号粗体页面标题,切换带 y 轴 32px 淡入动画)+ 内容区;移动端 Dock 变为底部固定悬浮图标栏。
- **Dock 导航**:图标按钮 `size-10/12 rounded-2xl`,带 `active 指示滑块`(absolute 定位的 `bg-sidebar-primary` 圆角方块,translate 过渡),hover 有 `scale-110` / active `scale-95` 微缩放,图标 `strokeWidth={2}`。
- **卡片**:`rounded-3xl border border-border bg-card p-4`,点击可放大为交互式详情弹窗(`morphing-dialog` 风格,容器 `rounded-3xl max-h-[90vh] overflow-y-auto`)。
- **指标块**:内嵌 `rounded-2xl border border-border/70 bg-background/80 p-2`,左侧小图标(`size-3.5`,语义色:成功 `text-emerald-500`,失败 `text-destructive`,常规 `text-primary`),适配 grid/list 两种形态。
- **图标徽标块**:`h-10 w-10 rounded-lg bg-primary/10 text-primary` 居中图标 + 左侧 label + 右侧数值。
- **数据展示**:数字使用 AnimatedNumber 动画;大数据量用 VirtualizedGrid(虚拟列表);图表用 recharts 统一色板取 `--chart-1..5`。
- **Toast**:`sonner`,`position="top-left"`,success/error/warning 图标分别着 `text-primary` / `text-destructive` 色。
- **加载动画**:index.html 内置 SVG 线稿路径描边动画(逐路径 `stroke-dashoffset` 分段绘制 + 整体呼吸淡出),随表单/页面切换淡出。

### 3.5 微交互与动效约定

- 页面内容切换用 `AnimatePresence mode="wait"` + y 轴 32px 位移 + opacity,`duration: 0.3`。
- 按钮/图标 hover `scale-110`,active/press `scale-95`,过渡 `duration-150 ease-linear`。
- 导航切换时 DnD 播报用 `flushSync` + `getBoundingClientRect()` 保证滑块首帧定位。
- 全局隐藏滚动条(`scrollbar-width:none` / `-ms-overflow-style:none`),内容区 `overscroll-contain`。

## 四、与 Go 后端的集成方式

- `vite.config.ts`:`build.outDir` 指向 `../static/out`,`emptyOutDir: true`,`base: './'`(相对路径)。
- Go 端用 embed/静态文件服务托管 `static/out`(参考 `static/static.go`),Go 单二进制部署前后端。
- dev 模式下 Vite `server.proxy` 把 `/api` 转发到 `127.0.0.1:8080`(Go 服务端口)。
- API 层 `src/api/` 每个资源一个文件,统一从 `queries.ts` 导出 useQuery/useMutation,错误统一 toast。

## 五、验收标准

1. 颜色、圆角、阴影、光斑与上述 oklch 色板一致,light/dark 两套都正常。
2. 桌面/移动端响应式,Dock 分别呈现为侧栏/底栏。
3. `pnpm dev` 可启动并代理 /api;`pnpm build` 产物进入 Go 的 static/out。
4. 三语(en/zh_hans/zh_hant)国际化结构就位。