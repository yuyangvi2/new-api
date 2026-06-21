# DESIGN.md — 素白 Monochrome Executive Design System v3.0

> 以白为底，以墨为字，以蓝为引。极简灰阶 + 克制电光蓝 + 重粗无衬线 + 大面积留白。
> 为 toB 产品打造的专业、可信、冷静的视觉语言。

---

## 1. Design Philosophy

- **灰阶为本**：界面主体由中性灰阶构成，黑墨承字、白底承内容，色彩不喧宾夺主
- **蓝为点睛**：电光蓝是「香料」不是「主菜」——只用于链接、焦点、小高亮；主操作用黑墨
- **重粗标题**：超粗无衬线（Inter 800）+ 紧字距，标题即排版的视觉重心
- **细线留白**：用 1px 细线分隔代替卡片描边与阴影，大面积留白营造高端克制感
- **4px Rhythm**：所有间距、字号、尺寸遵循 4px 网格

---

## 2. Color Tokens

### 2.1 灰阶 Gray — 中性骨架（冷中性灰）

| Token | Hex | Usage |
|-------|-----|-------|
| `--gray-0` | `#FFFFFF` | **页面主背景**、卡片底 |
| `--gray-50` | `#FAFAFB` | 侧边栏 / 面板底色，最浅表面 |
| `--gray-100` | `#F5F5F6` | 次级表面、悬停底色、代码 inline 底 |
| `--gray-200` | `#EAEAEC` | **默认分隔线 / 描边** |
| `--gray-300` | `#D8D8DC` | 强描边、Input 边框、次级按钮边框 |
| `--gray-400` | `#B8B9C0` | Input hover 边框、placeholder |
| `--gray-500` | `#9B9CA4` | Caption / Overline / 元信息文字 |
| `--gray-600` | `#71727B` | 辅助说明文字（Body SM 默认） |
| `--gray-700` | `#4E4F58` | — |
| `--gray-800` | `#33343A` | 次级文字、导航项 |
| `--gray-900` | `#1B1C20` | — |
| `--gray-950` | `#0A0A0B` | **正文主色 / 黑墨**、Primary 按钮、代码块底 |

### 2.2 墨色 Ink — 文字语义别名

| Token | Hex (= Gray) | Usage |
|-------|-----|-------|
| `--ink-primary` | `#0A0A0B` (950) | **正文主色**、标题、Primary 按钮背景 |
| `--ink-secondary` | `#33343A` (800) | 次级文字、标签、导航项 |
| `--ink-tertiary` | `#6B6C75` | 辅助说明文字、Body SM |
| `--ink-muted` | `#9B9CA4` (500) | Caption、Overline、元信息 |
| `--ink-faint` | `#B8B9C0` (400) | Input placeholder、最浅文字 |

> 用 `--ink-primary` (`#0A0A0B`) 替代纯黑 `#000000`——近黑带极微暖度，更耐看。

### 2.3 靛蓝 Accent — 主色调（克制电光蓝，`--accent-*`）

| Token | Hex | Usage |
|-------|-----|-------|
| `--accent-50` | `#EEF0FF` | Accent 最浅表面、Badge accent 背景、focus 光晕 |
| `--accent-100` | `#E0E3FF` | — |
| `--accent-200` | `#C2C8FF` | — |
| `--accent-300` | `#97A1FF` | 代码块 keyword 色（dark） |
| `--accent-400` | `#5E6CFF` | Hover 态、次强调 |
| `--accent-500` | `#2D4BFF` | **主强调色**：链接、focus border、小高亮、Accent 按钮 |
| `--accent-600` | `#1E37D6` | 行内代码文字色、链接 hover、Accent 按钮 hover |
| `--accent-700` | `#1629A6` | Badge accent 文字色、Info dark |
| `--accent-800` | `#111E78` | — |
| `--accent-900` | `#0C154D` | — |

> ⚠️ 蓝色是点睛色，不是主色面。**Primary 操作按钮用黑墨 `--ink-primary`，不用蓝。** 蓝色只出现在链接、焦点环、小标签、代码关键字等小面积场景。

### 2.4 语义色 Semantic（专业克制）

| Semantic | Light | Base | Dark |
|----------|-------|------|------|
| Success | `#ECF6F0` | `#18794E` | `#0B5437` |
| Warning | `#FCF4E6` | `#B8770C` | `#7A4E06` |
| Error | `#FCEFEF` | `#C8372D` | `#8E211A` |
| Info | `#EEF0FF` | `#2D4BFF` | `#1629A6` |

### 2.5 代码块语法高亮 Code Syntax（Shiki）

代码块底色统一 `--gray-950` (`#0A0A0B`)，**明亮 / 黑暗模式使用同一套高亮主题**，保证 token 对比度。配色克制——以近白承载主体，仅用 蓝 / 绿 / 金 三色点缀。

| Token 类型 | Hex | 说明 |
|-----------|-----|------|
| 默认文字 Default | `#D4D4D8` | 普通标识符、文本 |
| 注释 Comment | `#6A6B73` | 斜体，弱化 |
| 关键字 / 存储 / 运算符 Keyword | `#8FA3FF` | 软蓝：`export`、`const`、`def`、标签名 |
| 函数 / 类型 Function/Type | `#B8C2E8` | 浅蓝：函数调用、类型 |
| 字符串 String | `#9DD8B4` | 软绿：`"sk-..."` |
| 数字 / 常量 / 属性 Constant | `#E0C98A` | 软金：数值、`true`/`false`、属性符号 |
| 变量 / 参数 / 环境变量 Variable | `#E0C98A` | 软金，高对比：`OPENAI_API_KEY`、`model` |
| 标点 / 括号 Punctuation | `#71727B` | 弱化结构符号 |

> 行内代码（非代码块）：`--accent-600` (`#1E37D6`) 蓝字 + `--gray-100` 底，跟随明 / 暗模式切换。

---

## 3. Typography

### 3.1 字体栈

| Token | Font Stack | Usage |
|-------|-----------|-------|
| `--font-sans` | `'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif` | Display、全部标题、Body、Caption、Overline |
| `--font-mono` | `'JetBrains Mono', 'Fira Code', 'Consolas', monospace` | Overline / 标签、Badge、Code、数据展示、元信息 |

> 本系统**不使用衬线体**——全部 Inter，靠字重与字距拉开层级。Overline、section label、状态标签、代码统一用等宽体强化「工程 / 数据」气质。

### 3.2 字号层级

| Level | Font | Size | Weight | Line-height | Letter-spacing | Color Token |
|-------|------|------|--------|-------------|----------------|-------------|
| Display | sans | 64px (4rem) | 800 | 1.02 | -0.04em | `--ink-primary` |
| H1 | sans | 40px (2.5rem) | 800 | 1.1 | -0.035em | `--ink-primary` |
| H2 | sans | 30px (1.875rem) | 700 | 1.2 | -0.025em | `--ink-primary` |
| H3 | sans | 22px (1.375rem) | 700 | 1.3 | -0.015em | `--ink-primary` |
| H4 | sans | 18px (1.125rem) | 600 | 1.4 | -0.01em | `--ink-primary` |
| Body LG | sans | 18px (1.125rem) | 400 | 1.7 | 0 | `--ink-secondary` |
| Body | sans | 16px (1rem) | 400 | 1.7 | 0 | `--ink-secondary` |
| Body SM | sans | 14px (0.875rem) | 400 | 1.6 | 0 | `--ink-tertiary` |
| Caption | sans | 12px (0.75rem) | 500 | 1.5 | 0 | `--ink-muted` |
| Overline | mono | 11px (0.6875rem) | 500 | 1.5 | 0.1em, uppercase | `--ink-muted` |
| Code | mono | 14px (0.875rem) | 400 | 1.7 | 0 | `--accent-600` |

### 3.3 规则

- 标题全部无衬线，靠 **800 / 700 重字重 + 负字距**建立视觉重心
- 正文用 `--ink-secondary`，不用纯黑；辅助文字降到 `--ink-tertiary`
- Overline / section label / 状态标签用**等宽体大写 + 宽字距**
- 行高不低于 1.5，正文推荐 1.7

---

## 4. Spacing

基础单位 **4px**，所有间距必须是 4 的倍数。极简风格偏好更大的区块留白。

| Token | Value | Usage |
|-------|-------|-------|
| `--space-1` | 4px | 微间距：图标与文字间隙 |
| `--space-2` | 8px | 组件内紧凑间距：Badge padding |
| `--space-3` | 12px | 组件内间距：Input 横向 padding |
| `--space-4` | 16px | 组件内间距：Button padding、组件行 gap |
| `--space-5` | 20px | 组件间间距 |
| `--space-6` | 24px | Card 内边距、组件间间距 |
| `--space-8` | 32px | 区块间距 |
| `--space-10` | 40px | 区块间距 |
| `--space-12` | 48px | 页面区域间距 |
| `--space-16` | 64px | 页面级间距、首屏 padding |
| `--space-20` | 80px | 区块大间距 |
| `--space-24` | 96px | 首屏留白、section 间距（极简留白） |

### 规则

- 组件内：`space-1` ~ `space-4`
- 组件间：`space-5` ~ `space-8`
- 区块间：`space-12` ~ `space-20`
- 页面级：`space-20` ~ `space-24`（极简风格刻意放大）

---

## 5. Border Radius

极简风格偏好**更小更锐**的圆角。

| Token | Value | Usage |
|-------|-------|-------|
| `--radius-sm` | 6px | Checkbox、Badge、小元素 |
| `--radius-md` | 8px | **默认**：Button、Input、Tooltip |
| `--radius-lg` | 12px | Card、代码块、Toast |
| `--radius-xl` | 16px | 大型容器 |
| `--radius-full` | 9999px | Toggle、Avatar、Progress、Pill badge |

---

## 6. Shadows

极简风格**默认不用阴影**，仅用 1px 细线分隔。阴影只在浮层（下拉、Toast、Modal、代码块）出现，且极轻。基于中性黑 `rgba(10,10,11,...)`。

| Token | Value | Usage |
|-------|-------|-------|
| `--shadow-sm` | `0 1px 2px rgba(10,10,11,0.05)` | 微浮起：Input focus、Checkbox |
| `--shadow-md` | `0 4px 16px rgba(10,10,11,0.06), 0 1px 2px rgba(10,10,11,0.04)` | Card hover、Dropdown |
| `--shadow-lg` | `0 12px 32px rgba(10,10,11,0.08), 0 2px 8px rgba(10,10,11,0.04)` | Toast、代码块、Popover |
| `--shadow-xl` | `0 24px 56px rgba(10,10,11,0.12), 0 4px 12px rgba(10,10,11,0.04)` | Modal、大型浮层 |

---

## 7. Motion

| Token | Value | Usage |
|-------|-------|-------|
| `--ease-out` | `cubic-bezier(0.16, 1, 0.3, 1)` | **全局默认缓动**——弹性出场 |
| `--duration-fast` | 120ms | 悬停、边框色变化、toggle |
| `--duration-normal` | 200ms | 卡片悬停、进度条动画 |
| `--duration-slow` | 350ms | 页面过渡、大型位移 |

---

## 8. Component Specifications

### 8.1 Button

| Property | Default | Small | Large |
|----------|---------|-------|-------|
| Height | 40px | 32px | 48px |
| Padding | 12px 24px | 6px 14px | 14px 28px |
| Font size | 15px | 13px | 16px |
| Font weight | 600 | 600 | 600 |
| Border radius | 8px (`--radius-md`) | 8px | 8px |
| Border | 1px solid | 1px solid | 1px solid |
| Transition | 150ms ease-out | | |

**Variants:**

| Variant | Background | Text | Border | Hover |
|---------|-----------|------|--------|-------|
| Primary | `--ink-primary` | white | `--ink-primary` | bg → `#26272D` |
| Secondary | `--gray-0` | `--ink-primary` | `--gray-300` | bg → `--gray-100` |
| Accent | `--accent-500` | white | `--accent-500` | bg → `--accent-600` |
| Ghost | transparent | `--ink-secondary` | transparent | bg → `--gray-100` |
| Link | transparent | `--accent-600` | none | underline |
| Danger | `--error` | white | `--error` | bg → `--error-dark` |

> Primary = 黑墨；蓝色仅用于 Accent / Link 这类点睛场景。

### 8.2 Input

| Property | Value |
|----------|-------|
| Padding | 10px 12px |
| Border | 1px solid `--gray-300` |
| Border radius | 8px (`--radius-md`) |
| Background | `--gray-0` |
| Font | Inter, 14px, `--ink-primary` |
| Placeholder color | `--ink-faint` |
| Hover border | `--gray-400` |
| Focus border | `--accent-500` |
| Focus ring | `0 0 0 3px var(--accent-50)` |
| Error border | `--error` |
| Label | 13px, weight 600, `--ink-secondary` |
| Hint | 12px, `--ink-muted` |

### 8.3 Card

极简风格优先用 **1px 细线 + 留白**，而非阴影盒子。两种形态：

| Property | Bordered | Divided（分隔式） |
|----------|----------|------------------|
| Background | `--gray-0` | transparent |
| Border | 1px solid `--gray-200` | 仅用 1px `--gray-200` 竖/横线分隔 |
| Border radius | 10px (`--radius-lg`) | 0 |
| Padding | 24px (`--space-6`) | 24px |
| Shadow | none | none |
| Hover | border → `--gray-300`，`--shadow-md` 微浮 | bg → `--gray-50` |
| Transition | 200ms ease-out | 120ms |

**Card internal elements:**
- Overline: mono, 11px, weight 500, `--ink-muted`, uppercase, letter-spacing 0.1em
- Title: sans, 17px, weight 700, `--ink-primary`, letter-spacing -0.01em
- Body: 14px, `--ink-tertiary`, line-height 1.6
- “了解更多” link: 13px, weight 600, `--accent-600`

### 8.4 Badge

| Property | Value |
|----------|-------|
| Padding | 4px 10px |
| Border radius | 6px (`--radius-sm`) 或 full（pill） |
| Font size | 11.5px |
| Font weight | 600 |
| Letter spacing | 0.02em |

**Color variants:**

| Variant | Background | Text |
|---------|-----------|------|
| Ink（实心黑） | `--ink-primary` | white |
| Accent | `--accent-50` | `--accent-700` |
| Neutral（mono） | `--gray-100` | `--ink-tertiary`（等宽体） |
| Success | `--success-light` | `--success-dark` |
| Warning | `--warning-light` | `--warning-dark` |
| Error | `--error-light` | `--error-dark` |

### 8.5 Toggle

| Property | Value |
|----------|-------|
| Width / Height | 44px / 24px |
| Track radius | full |
| Thumb | 20px, white, `--shadow-sm` |
| Off background | `--gray-300` |
| On background | `--ink-primary`（黑墨，非蓝） |
| Transition | `--duration-fast` |

### 8.6 Checkbox

| Property | Value |
|----------|-------|
| Size | 18×18px |
| Border radius | 6px (`--radius-sm`) |
| Border | 1px solid `--gray-400` |
| Checked bg / border | `--ink-primary` |
| Checkmark | white |
| Transition | 120ms ease-out |

### 8.7 Avatar

| Size | Dimension | Font size |
|------|-----------|-----------|
| SM | 28×28px | 11px |
| Default | 40×40px | 14px |
| LG | 56×56px | 20px |

- Border radius: full（圆形）
- 背景：`--gray-200` + `--ink-secondary` 文字；或 `--ink-primary` + white
- Font weight: 600

### 8.8 Tooltip

- Background: `--ink-primary`
- Text: white, 12px, weight 500
- Padding: 6px 12px
- Border radius: 8px
- Position: above trigger, 8px gap

### 8.9 Toast

- Background: `--gray-0`
- Border: 1px solid `--gray-200`
- Border radius: 12px
- Shadow: `--shadow-lg`
- Padding: 12px 16px
- Title: 13px, weight 600, `--ink-primary`
- Description: 12px, `--ink-muted`

### 8.10 Progress Bar

| Property | Value |
|----------|-------|
| Height | 6px |
| Background | `--gray-200` |
| Border radius | full |
| Fill | `--ink-primary`（默认）/ `--accent-500`（强调） |
| Transition | `--duration-slow` `--ease-out` |

### 8.11 Divider

| Variant | Color |
|---------|-------|
| Light | `--gray-100` |
| Default | `--gray-200` |
| Strong | `--gray-300` |

---

## 9. Layout Patterns

### 9.1 Sidebar Navigation

- Width: 240px, fixed
- Background: `--gray-50`
- Border right: 1px solid `--gray-200`
- Padding: 32px 16px
- Nav items: 14px, weight 500, `--ink-secondary`
- Nav hover: bg `--gray-100`, color → `--ink-primary`
- Nav active: bg `--gray-100`, color → `--ink-primary`, weight 600，左侧 2px `--ink-primary` 指示条
- Nav border radius: 8px
- Group label: mono, 11px, weight 500, `--ink-muted`, uppercase, letter-spacing 0.12em

### 9.2 Main Content

- Max width: 800px（极简偏窄，提升阅读节奏）
- Padding: 64px 56px
- Section margin: 大留白（`--space-16` ~ `--space-20`）
- Section scroll-margin-top: 32px

### 9.3 Responsive

- ≤900px：隐藏 sidebar，main content 全宽，padding 缩至 24px

---

## 10. Code Implementation

### 10.1 CSS Custom Properties

```css
:root {
  /* 灰阶 Gray — 中性骨架 */
  --gray-0:   #FFFFFF;
  --gray-50:  #FAFAFB;
  --gray-100: #F5F5F6;
  --gray-200: #EAEAEC;
  --gray-300: #D8D8DC;
  --gray-400: #B8B9C0;
  --gray-500: #9B9CA4;
  --gray-600: #71727B;
  --gray-700: #4E4F58;
  --gray-800: #33343A;
  --gray-900: #1B1C20;
  --gray-950: #0A0A0B;

  /* 墨色 Ink — 文字语义 */
  --ink-primary:   #0A0A0B;
  --ink-secondary: #33343A;
  --ink-tertiary:  #6B6C75;
  --ink-muted:     #9B9CA4;
  --ink-faint:     #B8B9C0;

  /* 靛蓝 Accent — 克制电光蓝 */
  --accent-50:  #EEF0FF;
  --accent-100: #E0E3FF;
  --accent-200: #C2C8FF;
  --accent-300: #97A1FF;
  --accent-400: #5E6CFF;
  --accent-500: #2D4BFF;
  --accent-600: #1E37D6;
  --accent-700: #1629A6;
  --accent-800: #111E78;
  --accent-900: #0C154D;

  /* Semantic 语义色 */
  --success-light: #ECF6F0; --success: #18794E; --success-dark: #0B5437;
  --warning-light: #FCF4E6; --warning: #B8770C; --warning-dark: #7A4E06;
  --error-light:   #FCEFEF; --error:   #C8372D; --error-dark:   #8E211A;
  --info-light:    #EEF0FF; --info:    #2D4BFF; --info-dark:    #1629A6;

  /* Spacing */
  --space-1: 4px;  --space-2: 8px;  --space-3: 12px; --space-4: 16px;
  --space-5: 20px; --space-6: 24px; --space-8: 32px; --space-10: 40px;
  --space-12: 48px; --space-16: 64px; --space-20: 80px; --space-24: 96px;

  /* Radii */
  --radius-sm: 6px; --radius-md: 8px; --radius-lg: 12px;
  --radius-xl: 16px; --radius-full: 9999px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(10,10,11,0.05);
  --shadow-md: 0 4px 16px rgba(10,10,11,0.06), 0 1px 2px rgba(10,10,11,0.04);
  --shadow-lg: 0 12px 32px rgba(10,10,11,0.08), 0 2px 8px rgba(10,10,11,0.04);
  --shadow-xl: 0 24px 56px rgba(10,10,11,0.12), 0 4px 12px rgba(10,10,11,0.04);

  /* Typography */
  --font-sans: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif;
  --font-mono: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;

  /* Motion */
  --ease-out: cubic-bezier(0.16, 1, 0.3, 1);
  --duration-fast: 120ms;
  --duration-normal: 200ms;
  --duration-slow: 350ms;
}
```

### 10.2 Tailwind CSS Config

```js
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      colors: {
        gray: {
          0: '#FFFFFF', 50: '#FAFAFB', 100: '#F5F5F6', 200: '#EAEAEC',
          300: '#D8D8DC', 400: '#B8B9C0', 500: '#9B9CA4', 600: '#71727B',
          700: '#4E4F58', 800: '#33343A', 900: '#1B1C20', 950: '#0A0A0B',
        },
        accent: {
          50: '#EEF0FF', 100: '#E0E3FF', 200: '#C2C8FF', 300: '#97A1FF',
          400: '#5E6CFF', 500: '#2D4BFF', 600: '#1E37D6', 700: '#1629A6',
          800: '#111E78', 900: '#0C154D',
        },
        ink: {
          primary: '#0A0A0B', secondary: '#33343A', tertiary: '#6B6C75',
          muted: '#9B9CA4', faint: '#B8B9C0',
        },
      },
      fontFamily: {
        sans: ['Inter', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      borderRadius: { sm: '6px', md: '8px', lg: '12px', xl: '16px' },
      boxShadow: {
        sm: '0 1px 2px rgba(10,10,11,0.05)',
        md: '0 4px 16px rgba(10,10,11,0.06), 0 1px 2px rgba(10,10,11,0.04)',
        lg: '0 12px 32px rgba(10,10,11,0.08), 0 2px 8px rgba(10,10,11,0.04)',
        xl: '0 24px 56px rgba(10,10,11,0.12), 0 4px 12px rgba(10,10,11,0.04)',
      },
      transitionTimingFunction: { out: 'cubic-bezier(0.16, 1, 0.3, 1)' },
    },
  },
}
```

---

## 11. Anti-patterns (禁止)

- ❌ 不要用纯黑 `#000000` / 纯白文字——黑墨用 `--ink-primary` (`#0A0A0B`)
- ❌ **不要把蓝色当主色面铺开**——Primary 操作按钮用黑墨，蓝只用于链接 / 焦点 / 小标签 / 代码关键字
- ❌ 不要使用衬线体——本系统全 Inter，靠字重 + 字距分层
- ❌ 不要使用彩色渐变、霓虹辉光、毛玻璃——保持灰阶 + 单一蓝点睛
- ❌ 不要用粗重阴影做卡片——优先 1px 细线 + 留白，阴影仅留给浮层
- ❌ 不要使用暖色（terracotta / 朱砂 / 藤黄）——这是上一代「宣卷」风格，已废弃
- ❌ 不要使用 generic blue `#3B82F6`——统一用 `--accent-500` (`#2D4BFF`)
- ❌ 不要使用非 4px 倍数的间距值
- ❌ 不要使用 `ease-in-out`——统一用 `--ease-out` (`cubic-bezier(0.16, 1, 0.3, 1)`)
- ❌ 标题字重不要低于 700——重粗是本系统的标志
