# GitHub仓库设置和自动更新配置指南

## 🚀 第一步：创建GitHub仓库

### 1. 在GitHub上创建新仓库

1. 访问 [https://github.com/new](https://github.com/new)
2. 仓库名称：`url-navigator`
3. 描述：`A modern desktop application for managing website bookmarks and URL navigation`
4. 设为 Public（用于GitHub Releases）
5. 不要初始化README、.gitignore或license（我们已经有了）
6. 点击 "Create repository"

### 2. 连接本地仓库到GitHub

在项目根目录执行以下命令（替换为您的GitHub用户名）：

```bash
# 添加远程仓库
git remote add origin https://github.com/YOUR_GITHUB_USERNAME/url-navigator.git

# 推送代码到GitHub
git branch -M main
git push -u origin main
```

## 🔧 第二步：配置真实更新功能

### 1. 更新GitHub仓库信息

编辑 `updater.go` 文件，将以下常量替换为您的实际信息：

```go
const (
    CurrentVersion = "1.0.0"
    // 替换为您的GitHub信息
    GitHubOwner = "YOUR_GITHUB_USERNAME"  // 您的GitHub用户名
    GitHubRepo  = "url-navigator"         // 仓库名称
)
```

### 2. 示例配置

假设您的GitHub用户名是 `johndoe`，则配置应该是：

```go
const (
    CurrentVersion = "1.0.0"
    GitHubOwner = "johndoe"
    GitHubRepo  = "url-navigator"
)
```

这样更新检查将调用：`https://api.github.com/repos/johndoe/url-navigator/releases/latest`

## 📦 第三步：创建首个发布版本

### 1. 构建应用程序

```bash
# 构建应用
wails build

# 或者构建压缩版本
wails build -compress
```

### 2. 创建Git标签

```bash
# 创建版本标签
git tag v1.0.0

# 推送标签到GitHub
git push origin v1.0.0
```

### 3. 创建GitHub Release

1. 访问您的仓库：`https://github.com/YOUR_USERNAME/url-navigator`
2. 点击 "Releases" 选项卡
3. 点击 "Create a new release"
4. 配置发布信息：
   - **Tag version**: `v1.0.0`
   - **Release title**: `URL Navigator v1.0.0`
   - **Description**:
     ```markdown
     # URL Navigator v1.0.0

     首个正式版本发布！

     ## 功能特性
     - 🔖 网址书签管理
     - 📁 分类系统
     - 🔍 智能搜索
     - 🚀 自动更新功能
     - 🎨 现代化UI界面

     ## 下载
     请下载适合您操作系统的版本：
     - Windows: URLNavigator.exe
     - macOS: URLNavigator.app
     - Linux: URLNavigator
     ```

5. 上传构建的文件：
   - 将 `build/bin/URLNavigator.exe` 重命名为 `URLNavigator.exe` 并上传

6. 点击 "Publish release"

## 🔄 第四步：启用自动构建（可选）

### GitHub Actions自动构建

我们已经创建了 `.github/workflows/build.yml` 文件，它会：

1. **自动触发**：当您推送新的版本标签时
2. **多平台构建**：Windows、macOS、Linux
3. **自动发布**：构建完成后自动创建GitHub Release

### 使用自动构建：

```bash
# 更新版本号（在wails.json中）
# 然后提交更改
git add .
git commit -m "bump version to 1.1.0"

# 创建新标签
git tag v1.1.0
git push origin v1.1.0

# GitHub Actions会自动构建并发布
```

## 🧪 第五步：测试更新功能

### 1. 更新代码中的仓库信息

确保 `updater.go` 中的信息正确：

```go
const (
    CurrentVersion = "1.0.0"
    GitHubOwner = "YOUR_ACTUAL_GITHUB_USERNAME"
    GitHubRepo  = "url-navigator"
)
```

### 2. 重新构建应用

```bash
# 重新生成绑定
wails build

# 启动应用测试
./build/bin/URLNavigator.exe
```

### 3. 测试真实更新

1. 点击应用中的 "检查更新" 按钮
2. 应该能连接到您的GitHub仓库检查版本
3. 如果有新版本，会提示下载和安装

### 4. 模拟新版本测试

1. 创建一个更高版本号的测试发布 (如 v1.0.1)
2. 运行v1.0.0版本的应用
3. 检查更新应该会检测到新版本

## 📋 配置检查清单

- [ ] GitHub仓库已创建
- [ ] 代码已推送到GitHub
- [ ] `updater.go` 中的GitHub信息已更新
- [ ] 首个Release已创建并上传了可执行文件
- [ ] 应用可以成功检查更新
- [ ] GitHub Actions工作流正常运行

## 🔒 安全考虑

1. **版本验证**：生产环境建议添加签名验证
2. **下载验证**：可以添加校验和验证
3. **HTTPS**：始终使用HTTPS进行更新下载
4. **权限**：确保应用有权限替换自身

## 🎯 下一步

完成上述配置后，您的URL Navigator应用就具备了完整的自动更新功能：

1. **检查更新**：连接GitHub Releases API
2. **版本比较**：智能版本号比较
3. **自动下载**：下载对应平台的新版本
4. **安全更新**：使用selfupdate库安全替换
5. **回滚机制**：更新失败时自动回滚

您的用户将能够一键更新到最新版本！🎉