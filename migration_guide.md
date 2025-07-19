# 版本管理和自动更新系统迁移指南

## 问题总结

### 原有问题
1. **版本信息硬编码**: 版本号分散在多个文件中，需要手动同步
2. **GitHub仓库信息未配置**: 使用占位符导致API调用失败
3. **自动更新无法工作**: 由于配置问题导致更新检测失败
4. **发布流程繁琐**: 需要手动修改多个文件并同步版本号

### 解决方案概述
- ✅ 统一版本管理：支持编译时注入和配置文件两种方式
- ✅ 动态GitHub仓库配置：支持多种配置方式
- ✅ 增强错误处理：提供详细的错误信息和回退机制
- ✅ 自动化构建流程：一键构建和发布

## 迁移步骤

### 1. 替换文件

#### 1.1 创建新的版本管理文件
```bash
# 创建 version.go 文件
cp version_manager.go version.go
```

#### 1.2 替换 updater.go
```bash
# 备份原文件
mv updater.go updater.go.backup

# 使用新的 updater
cp updater_improved.go updater.go
```

#### 1.3 更新 wails.json
```bash
# 备份原配置
cp wails.json wails.json.backup

# 使用新配置
cp updated_wails_config.json wails.json
```

#### 1.4 更新构建脚本
```bash
# 备份原脚本
mv release.sh release.sh.backup

# 使用新的构建脚本
cp build_release.sh release.sh
chmod +x release.sh
```

### 2. 配置 GitHub 仓库信息

#### 方法一：更新 wails.json（推荐）
```json
{
  "github": {
    "owner": "您的GitHub用户名",
    "repo": "url-navigator"
  }
}
```

#### 方法二：使用环境变量（构建时）
```bash
export GITHUB_OWNER="您的GitHub用户名"
export GITHUB_REPO="url-navigator"
```

#### 方法三：创建用户配置文件
```bash
# 在应用数据目录创建 version.json
# Windows: %APPDATA%\.urlnavigator\version.json
```

### 3. 修改 app.go

在 `app.go` 的 `NewApp()` 或应用初始化函数中添加：

```go
// 在 NewApp 函数中添加
func NewApp() *App {
    app := &App{}
    
    // 初始化版本信息
    dataDir, _ := app.GetDataDir()
    if err := InitVersionInfo(dataDir); err != nil {
        // 记录错误但不阻止应用启动
        fmt.Printf("警告: 版本信息初始化失败: %v\n", err)
    }
    
    return app
}
```

### 4. 测试新系统

#### 4.1 测试版本获取
```bash
# 构建并运行应用
wails dev

# 在应用中调用
GetCurrentVersion()
GetVersionInfo()
```

#### 4.2 测试更新检查
```bash
# 确保已配置 GitHub 信息后
CheckForUpdates()

# 使用测试函数
TestUpdateAvailable()
TestNoUpdate()
```

### 5. 使用新的构建流程

#### 5.1 开发构建
```bash
# 开发模式
wails dev

# 普通构建
wails build
```

#### 5.2 发布构建
```bash
# 自动构建和发布
./release.sh 1.3.0

# 仅构建不发布
./release.sh 1.3.0 --skip-release

# 仅发布不构建（需要已有构建文件）
./release.sh 1.3.0 --skip-build
```

## 新功能特性

### 版本管理
- **多源版本信息**: 支持编译时注入、配置文件、wails.json
- **自动版本同步**: 构建时自动同步所有文件的版本号
- **版本验证**: 自动验证版本号格式和递增

### 自动更新
- **智能仓库检测**: 自动从 Git remote 获取仓库信息
- **详细错误信息**: 提供具体的错误原因和解决建议
- **多文件类型支持**: 智能查找 Windows 可执行文件
- **安全回退**: 更新失败时自动回滚

### 构建流程
- **一键构建**: 版本更新、构建、发布一步完成
- **环境检查**: 自动检查必要工具和依赖
- **GitHub 集成**: 支持 GitHub CLI 自动创建 Release
- **灵活参数**: 支持跳过构建或发布步骤

## 故障排除

### 版本信息获取失败
```bash
# 检查配置文件
cat %APPDATA%\.urlnavigator\version.json

# 检查 wails.json
grep -A 5 "github" wails.json
```

### 更新检查失败
1. **GitHub 仓库未找到 (404)**
   - 检查仓库名称是否正确
   - 确认仓库是公开的
   - 验证是否有 Release

2. **网络连接失败**
   - 检查网络连接
   - 验证防火墙设置
   - 尝试手动访问 GitHub API

3. **API 限制 (403/429)**
   - 等待后重试
   - 考虑使用 GitHub Token

### 构建失败
```bash
# 检查 Wails 环境
wails doctor

# 检查前端依赖
cd frontend && yarn install

# 检查 Go 模块
go mod tidy
```

## 最佳实践

### 开发流程
1. 开发功能时使用 `wails dev`
2. 测试时使用测试更新函数
3. 发布前先测试构建: `wails build`
4. 使用自动化脚本发布: `./release.sh X.Y.Z`

### 版本管理
1. 遵循语义化版本控制 (SemVer)
2. 在 CHANGELOG 中记录版本变更
3. 每次发布前测试自动更新功能
4. 保持版本号在所有文件中同步

### 安全考虑
1. 不要在代码中硬编码敏感信息
2. 使用 HTTPS 下载更新
3. 验证下载文件的完整性
4. 提供用户确认更新的选项

## 技术细节

### 版本获取优先级
1. 编译时注入的版本信息 (ldflags)
2. wails.json 中的版本和 GitHub 信息
3. 用户配置文件 (version.json)
4. 硬编码的默认值

### 更新检测流程
1. 检查运行平台 (仅 Windows 支持)
2. 验证 GitHub 配置信息
3. 调用 GitHub API 获取最新 Release
4. 比较版本号确定是否需要更新
5. 查找合适的下载文件

### 构建注入参数
```bash
-ldflags "-X main.Version=1.3.0 -X main.GitHubOwner=username -X main.GitHubRepo=repo"
```

这个迁移指南应该能帮助您从原有的硬编码版本系统迁移到新的动态版本管理系统。