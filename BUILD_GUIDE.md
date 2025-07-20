# URL Navigator 构建指南

本文档提供URL Navigator项目的构建、开发和发布指南。

## 系统要求

- Go 1.20+
- Node.js 16+
- Yarn 包管理器
- Wails v2.10+
- Git（用于发布流程）

## 开发工具

项目现在使用统一的Go工具来处理所有开发、构建和发布任务：

```bash
# 查看帮助
go run tools/urlnav.go help

# 开发模式 (热重载)
go run tools/urlnav.go dev

# 构建应用
go run tools/urlnav.go build

# 运行应用
go run tools/urlnav.go run

# 发布新版本
go run tools/urlnav.go release v1.4.0
```

## 版本格式

- **主要格式**: `vX.Y.Z` (例如: `v1.3.0`)
- **备选格式**: `X.Y.Z` (工具会自动添加`v`前缀)
- 示例: `v1.3.0`, `v2.0.1`, `v1.2.10`

## 配置管理

所有配置通过`version.json`作为**稳定模板**管理：

> **重要**: `version.json`作为配置模板，在发布过程中**不会被修改**。版本号通过工具参数管理，只应用到`wails.json`和`frontend/package.json`。

```json
{
  "version": "1.2.1",  // ⚠️ 模板版本 - 不会被脚本更新
  "github": {
    "owner": "wangyaxings",
    "repo": "url-navigator"
  },
  "app": {
    "name": "URLNavigator",
    "display_name": "URL Navigator",
    "description": "A beautiful URL bookmark manager with auto-update functionality"
  },
  "build": {
    "platform": "windows/amd64",
    "flags": ["-tags", "production", "-trimpath", "-clean"],
    "ldflags": ["-H=windowsgui", "-s", "-w"]
  },
  "release": {
    "create_github_release": true,
    "auto_open_browser": true,
    "commit_message_template": "chore: bump version to {version}",
    "tag_message_template": "Release {version}"
  }
}
```

## 开发流程

### 开发模式

启动带热重载的开发服务器：

```bash
go run tools/urlnav.go dev
```

这将：
- 自动安装前端依赖
- 启动Wails开发服务器
- 提供热重载功能
- 按Ctrl+C停止

### 构建应用

构建生产版本：

```bash
go run tools/urlnav.go build
```

这将：
- 安装/更新前端依赖
- 构建前端静态文件
- 编译Wails应用
- 生成`build/bin/URLNavigator.exe`

### 运行应用

运行构建的应用：

```bash
go run tools/urlnav.go run
```

如果应用未构建，会自动先执行构建。

## 发布流程

### 版本发布

发布新版本的完整流程：

```bash
# 完整发布流程
go run tools/urlnav.go release v1.4.0

# 只构建，不推送到Git
go run tools/urlnav.go release v1.4.0 -skip-release

# 只推送，不重新构建
go run tools/urlnav.go release v1.4.0 -skip-build

# 强制执行，跳过确认
go run tools/urlnav.go release v1.4.0 -force
```

### 发布选项

- `-skip-build`: 跳过构建过程
- `-skip-release`: 跳过Git操作（提交、标签、推送）
- `-force`: 强制执行，跳过所有确认

### 发布检查列表

发布工具会自动执行以下检查：

1. ✅ **版本格式验证**: 确保版本号格式正确
2. ✅ **Git状态检查**: 确保工作目录干净
3. ✅ **依赖检查**: 验证所需工具可用
4. ✅ **版本文件更新**: 同步更新配置文件
5. ✅ **构建验证**: 确保应用构建成功
6. ✅ **Git操作**: 提交、标签、推送

### 版本管理逻辑

发布工具按以下优先级管理版本：

1. **wails.json** - 实际版本源（动态更新）
2. **frontend/package.json** - 前端版本（同步更新）
3. **version.json** - 配置模板（保持不变）

## 文件结构

### 关键文件

- `tools/urlnav.go` - 统一开发和发布工具
- `main.go` - 应用程序入口点
- `app.go` - 主要应用逻辑和Wails绑定
- `version.go` - 版本管理逻辑
- `wails.json` - Wails配置和当前版本
- `version.json` - 发布配置模板

### 前端结构

- `frontend/src/components/` - React组件
- `frontend/src/services/` - 服务层
- `frontend/src/types/` - TypeScript类型定义
- `frontend/wailsjs/` - 自动生成的Go绑定

## 故障排除

### 常见问题

1. **前端构建失败**
   ```bash
   cd frontend
   yarn install
   yarn build
   ```

2. **Wails绑定问题**
   ```bash
   wails generate module
   ```

3. **Git推送失败**
   - 检查网络连接
   - 验证远程仓库权限
   - 使用`-skip-release`选项进行本地构建

### 环境验证

运行以下命令验证环境：

```bash
# 检查Go
go version

# 检查Node.js和Yarn
node --version
yarn --version

# 检查Wails
wails version

# 检查Git
git --version
```

## 部署

### GitHub Actions

项目配置了GitHub Actions自动化：

1. **推送标签**触发构建
2. **自动创建Release**
3. **上传构建产物**
4. **触发应用内更新**

### 手动部署

如果需要手动部署：

1. 运行完整发布流程
2. 检查GitHub Actions状态
3. 验证Release创建
4. 测试应用内更新

## 最佳实践

1. **开发时**使用`go run tools/urlnav.go dev`
2. **测试前**使用`go run tools/urlnav.go build`验证构建
3. **发布前**确保所有更改已提交
4. **使用语义化版本**号（v1.2.3）
5. **测试版本**使用`-skip-release`选项

---

## 新功能亮点

### 🚀 统一的Go工具

- 替换了所有批处理文件和PowerShell脚本
- 提供一致的跨平台体验
- 彩色输出和清晰的进度指示

### 🔧 改进的版本管理

- 自动版本注入到构建中
- 保持配置模板稳定性
- 智能的版本读取优先级

### 📦 简化的发布流程

- 一键式发布命令
- 内置安全检查
- 灵活的跳过选项

如有问题，请参考[GitHub仓库](https://github.com/wangyaxings/url-navigator)或提交Issue。