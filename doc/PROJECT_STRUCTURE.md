# URL Navigator 项目结构文档

本文档详细说明了URL Navigator项目中每个文件和目录的作用。

## 📁 项目总览

URL Navigator是一个基于Wails框架的桌面应用程序，使用Go作为后端，React + TypeScript作为前端。

```
AutoUpdate/
├── 🔧 开发工具
│   └── tools/urlnav.go           # 统一的开发和发布工具
├── 🖥️ 后端代码 (Go)
│   ├── main.go                   # 应用程序入口点
│   ├── app.go                    # 主要应用逻辑和Wails绑定
│   ├── version.go                # 版本管理逻辑
│   └── updater.go                # 自动更新功能
├── 🎨 前端代码 (React + TypeScript)
│   └── frontend/                 # 前端项目目录
├── ⚙️ 配置文件
│   ├── wails.json                # Wails框架配置
│   ├── version.json              # 版本和构建配置模板
│   ├── go.mod/go.sum            # Go模块依赖
│   └── .gitignore               # Git忽略文件配置
├── 📚 文档
│   ├── README.md                 # 项目介绍和使用说明
│   ├── BUILD_GUIDE.md           # 构建和发布指南
│   ├── AUTO_UPDATE_GUIDE.md     # 自动更新功能说明
│   └── PROJECT_STRUCTURE.md     # 本文档
├── 🔒 许可证
│   └── LICENSE                   # MIT许可证
├── 🚀 构建产物
│   └── build/                    # 编译输出目录
└── 🔄 版本控制
    └── .git/                     # Git仓库
```

## 📂 详细文件说明

### 🔧 开发工具

#### `tools/urlnav.go`
**统一的开发和发布工具**
- **作用**: 替代所有批处理和PowerShell脚本的Go工具
- **功能**:
  - `dev` - 启动开发模式（热重载）
  - `build` - 构建生产版本
  - `run` - 运行构建的应用
  - `release` - 发布新版本（版本管理、Git操作）
- **特点**: 跨平台兼容、彩色输出、错误处理完善

### 🖥️ 后端代码 (Go)

#### `main.go`
**应用程序入口点**
- **作用**: Wails应用的启动文件
- **功能**:
  - 创建App实例
  - 配置Wails应用选项（窗口大小、背景色等）
  - 初始化版本信息
  - 绑定Go函数到前端

#### `app.go`
**主要应用逻辑和Wails绑定**
- **作用**: 核心业务逻辑，暴露给前端的API
- **功能**:
  - URL书签管理（增删改查、搜索、排序）
  - 分类管理
  - 数据导入导出（Chrome/Firefox/Edge书签）
  - 高级搜索功能
  - 文件操作和数据持久化
- **数据存储**: 使用JSON文件存储在用户目录 `~/.urlnavigator/`

#### `version.go`
**版本管理逻辑**
- **作用**: 处理应用版本信息的读取、管理和显示
- **功能**:
  - 多源版本读取（编译时注入 > wails.json > 用户配置 > 默认模板）
  - 版本信息初始化和缓存
  - 版本格式化和显示
  - 版本配置文件管理
- **设计**: 支持运行时版本切换和调试

#### `updater.go`
**自动更新功能**
- **作用**: 检查和下载应用更新
- **功能**:
  - GitHub Releases API集成
  - 版本比较和更新检查
  - 自动下载和应用更新
  - 更新进度显示
  - 更新失败回滚机制

### 🎨 前端代码 (React + TypeScript)

#### `frontend/`目录结构
```
frontend/
├── src/
│   ├── App.tsx                   # 主应用组件
│   ├── main.tsx                  # React应用入口
│   ├── index.css                 # 全局样式
│   ├── components/               # React组件
│   │   ├── ui/                   # 基础UI组件库
│   │   ├── CategoryManager.tsx   # 分类管理
│   │   ├── UpdateChecker.tsx     # 更新检查器
│   │   ├── URLFormDialog.tsx     # URL表单对话框
│   │   ├── VersionInfo.tsx       # 版本信息显示
│   │   ├── ImportExport.tsx      # 导入导出功能
│   │   ├── AdvancedSearch.tsx    # 高级搜索
│   │   ├── DraggableURLCard.tsx  # 可拖拽URL卡片
│   │   ├── URLListView.tsx       # URL列表视图
│   │   ├── ThemeToggle.tsx       # 主题切换
│   │   ├── LayoutControls.tsx    # 布局控制
│   │   ├── ContextMenu.tsx       # 右键菜单
│   │   └── KeyboardShortcutsHelp.tsx # 快捷键帮助
│   ├── services/
│   │   └── appService.ts         # 应用服务层
│   ├── contexts/
│   │   └── ThemeContext.tsx      # 主题上下文
│   ├── hooks/
│   │   └── useKeyboardShortcuts.ts # 快捷键Hook
│   ├── lib/
│   │   ├── utils.ts              # 工具函数
│   │   ├── cache.ts              # 缓存工具
│   │   └── cacheManager.ts       # 缓存管理器
│   └── types/
│       └── index.ts              # TypeScript类型定义
├── wailsjs/                      # Wails自动生成的绑定
│   ├── go/main/                  # Go函数绑定
│   └── runtime/                  # Wails运行时
├── dist/                         # 构建输出目录
├── node_modules/                 # npm依赖
├── package.json                  # npm配置和依赖
├── yarn.lock                     # Yarn锁文件
├── index.html                    # HTML模板
├── vite.config.ts               # Vite构建配置
├── tsconfig.json                # TypeScript配置
├── tsconfig.node.json           # Node.js TypeScript配置
├── tailwind.config.js           # Tailwind CSS配置
└── postcss.config.js            # PostCSS配置
```

#### 前端主要组件说明

**`App.tsx`** - 主应用组件
- 应用程序的根组件
- 状态管理和路由逻辑
- 组件组合和布局

**`components/ui/`** - 基础UI组件库
- 基于shadcn/ui的组件系统
- 包含: Button, Card, Dialog, Input, Badge等
- 统一的设计系统和样式

**`services/appService.ts`** - 应用服务层
- 封装Wails Go函数调用
- 版本信息管理API
- 错误处理和类型安全

### ⚙️ 配置文件

#### `wails.json`
**Wails框架配置**
- **作用**: Wails应用的主配置文件
- **内容**:
  - 应用元信息（名称、版本、描述）
  - 前端构建配置
  - 构建选项和编译参数
  - GitHub仓库信息
  - ldflags版本注入配置

#### `version.json`
**版本和构建配置模板**
- **作用**: 发布工具的配置模板（不会被自动修改）
- **内容**:
  - 默认版本号（模板）
  - GitHub仓库信息
  - 应用信息配置
  - 构建参数和标志
  - 发布流程配置

#### `go.mod` / `go.sum`
**Go模块依赖管理**
- **作用**: Go项目的依赖声明和版本锁定
- **内容**: Wails、系统交互、JSON处理等依赖

#### `.gitignore`
**Git忽略文件配置**
- **作用**: 指定Git不跟踪的文件和目录
- **内容**: 构建产物、依赖目录、临时文件等

### 📚 文档

#### `README.md`
**项目介绍和使用说明**
- 项目概述和特性介绍
- 快速开始指南
- 安装和使用说明
- 贡献指南

#### `BUILD_GUIDE.md`
**构建和发布指南**
- 开发环境配置
- 构建流程说明
- 发布流程和版本管理
- 故障排除指南

#### `AUTO_UPDATE_GUIDE.md`
**自动更新功能说明**
- 自动更新机制介绍
- 配置和使用方法
- GitHub集成说明

#### `PROJECT_STRUCTURE.md`
**项目结构文档（本文档）**
- 详细的文件和目录说明
- 架构概述和设计理念

### 🔒 许可证

#### `LICENSE`
**MIT许可证**
- 项目的开源许可证
- 使用权限和限制说明

### 🚀 构建产物

#### `build/`目录
**编译输出目录**
```
build/
├── bin/
│   └── URLNavigator.exe          # Windows可执行文件
├── windows/
│   └── [构建临时文件]
└── appicon.png                   # 应用图标
```

### 🎯 核心特性对应的文件

| 特性 | 主要文件 |
|------|----------|
| **书签管理** | `app.go`, `frontend/src/App.tsx` |
| **分类功能** | `app.go` (GetCategories, AddCategory), `frontend/src/components/CategoryManager.tsx` |
| **搜索功能** | `app.go` (SearchURLs, AdvancedSearchURLs), `frontend/src/components/AdvancedSearch.tsx` |
| **导入导出** | `app.go` (ImportChromeBookmarks, ExportBookmarks), `frontend/src/components/ImportExport.tsx` |
| **自动更新** | `updater.go`, `frontend/src/components/UpdateChecker.tsx` |
| **版本管理** | `version.go`, `frontend/src/components/VersionInfo.tsx` |
| **主题系统** | `frontend/src/contexts/ThemeContext.tsx`, `frontend/src/components/ThemeToggle.tsx` |
| **开发工具** | `tools/urlnav.go` |

## 🔄 数据流

### 前端 ↔ 后端通信
```
React Components → appService.ts → Wails绑定 → Go函数 → 返回结果
```

### 数据存储
```
Go应用 → JSON文件 → 用户目录 (~/.urlnavigator/)
├── urls.json        # 书签数据
├── categories.json  # 分类数据
└── version.json     # 用户版本配置
```

### 版本管理流
```
编译时注入 → wails.json → 用户配置 → 默认模板 → 显示版本
```

## 🛠️ 开发流程

1. **开发**: `go run tools/urlnav.go dev`
2. **构建**: `go run tools/urlnav.go build`
3. **测试**: `go run tools/urlnav.go run`
4. **发布**: `go run tools/urlnav.go release v1.x.x`

## 📝 维护说明

- **添加新功能**: 在`app.go`中添加Go函数，在前端添加对应组件
- **修改UI**: 主要在`frontend/src/components/`目录下工作
- **版本发布**: 使用统一的Go工具 `tools/urlnav.go`
- **依赖更新**: 前端使用`yarn`，后端使用`go mod`

---

*此文档随项目更新而维护，最后更新：2025年1月*