# 自动更新系统完整实现指南

> 本文档详细介绍了基于 Wails 框架的桌面应用自动更新系统的完整实现方案，可作为其他项目的参考模板和最佳实践。

## 📋 目录

- [系统架构](#系统架构)
- [核心组件](#核心组件)
- [实现详解](#实现详解)
- [版本管理](#版本管理)
- [问题与解决方案](#问题与解决方案)
- [最佳实践](#最佳实践)
- [部署指南](#部署指南)

## 🏗️ 系统架构

### 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    自动更新系统架构                          │
├─────────────────────────────────────────────────────────────┤
│  前端 (React + TypeScript)                                 │
│  ┌──────────────────┐  ┌─────────────────┐                  │
│  │  UpdateChecker   │  │  VersionInfo    │                  │
│  │  ・检查更新按钮   │  │  ・版本显示     │                  │
│  │  ・更新对话框     │  │  ・来源标识     │                  │
│  │  ・进度显示       │  │  ・调试信息     │                  │
│  └──────────────────┘  └─────────────────┘                  │
├─────────────────────────────────────────────────────────────┤
│  后端 (Go + Wails)                                         │
│  ┌──────────────────┐  ┌─────────────────┐                  │
│  │     updater.go   │  │    version.go   │                  │
│  │  ・GitHub API    │  │  ・版本管理     │                  │
│  │  ・下载管理      │  │  ・多源读取     │                  │
│  │  ・进度跟踪      │  │  ・来源追踪     │                  │
│  │  ・自动重启      │  │  ・兜底机制     │                  │
│  └──────────────────┘  └─────────────────┘                  │
└─────────────────────────────────────────────────────────────┘
```

### 核心流程

```
用户触发检查更新
        ↓
调用 GitHub API 获取最新版本
        ↓
版本号比较 (当前 vs 最新)
        ↓
    有新版本?
        ↓
显示更新对话框 (版本信息 + 更新说明)
        ↓
用户确认更新
        ↓
后台下载新版本 (显示进度)
        ↓
使用 selfupdate 应用更新
        ↓
自动重启应用
```

## 🧩 核心组件

### 1. 版本管理组件 (`version.go`)

**设计理念**：多源版本读取 + 透明来源追踪 + 兜底机制

```go
// 版本来源类型
type VersionSource string
const (
    SourceCompileTime VersionSource = "compile_time"   // 编译时注入 (最可靠)
    SourceWailsJSON   VersionSource = "wails_json"     // 配置文件
    SourceVersionJSON VersionSource = "version_json"   // 备份配置
    SourceDefault     VersionSource = "default"        // 兜底方案
)

// 增强的版本信息结构
type VersionInfo struct {
    Version     string        `json:"version"`
    GitHubOwner string        `json:"github_owner"`
    GitHubRepo  string        `json:"github_repo"`
    AppName     string        `json:"app_name"`
    Source      VersionSource `json:"source"`       // 🔍 版本来源
    IsDefault   bool          `json:"is_default"`   // ⚠️ 是否为默认值
}
```

**版本获取优先级**：
1. **编译时注入** (ldflags) - 最可靠
2. **wails.json配置** - 开发配置
3. **version.json配置** - 备份配置
4. **默认兜底值** - 防止unknown

### 2. 更新检查组件 (`updater.go`)

**核心功能**：
- GitHub API集成
- 版本比较算法
- 断点续传下载
- 进度实时跟踪
- 失败回滚机制

```go
// 更新信息结构
type UpdateInfo struct {
    HasUpdate      bool   `json:"hasUpdate"`
    CurrentVersion string `json:"currentVersion"`
    LatestVersion  string `json:"latestVersion"`
    UpdateURL      string `json:"updateUrl"`
    ReleaseNotes   string `json:"releaseNotes"`
    ErrorMessage   string `json:"errorMessage,omitempty"`
}

// 更新进度结构
type UpdateProgress struct {
    Phase          string `json:"phase"`          // downloading/installing/completed/error
    Progress       int    `json:"progress"`       // 0-100
    Speed          string `json:"speed"`          // 下载速度
    ETA            string `json:"eta"`            // 预计剩余时间
    Downloaded     int64  `json:"downloaded"`     // 已下载字节
    Total          int64  `json:"total"`          // 总字节
    Message        string `json:"message"`        // 状态消息
    Error          string `json:"error,omitempty"` // 错误信息
}
```

## 🔧 实现详解

### 版本比较算法

```go
func compareVersions(v1, v2 string) int {
    // 清理版本字符串
    v1 = strings.TrimPrefix(strings.TrimSpace(v1), "v")
    v2 = strings.TrimPrefix(strings.TrimSpace(v2), "v")

    if v1 == v2 {
        return 0
    }

    parts1 := strings.Split(v1, ".")
    parts2 := strings.Split(v2, ".")

    // 确保两个版本都有三个部分
    for len(parts1) < 3 {
        parts1 = append(parts1, "0")
    }
    for len(parts2) < 3 {
        parts2 = append(parts2, "0")
    }

    for i := 0; i < 3; i++ {
        num1Str := strings.Split(parts1[i], "-")[0]
        num2Str := strings.Split(parts2[i], "-")[0]

        num1, err1 := strconv.Atoi(num1Str)
        num2, err2 := strconv.Atoi(num2Str)

        if err1 != nil || err2 != nil {
            // 字符串比较
            if parts1[i] < parts2[i] {
                return -1
            } else if parts1[i] > parts2[i] {
                return 1
            }
            continue
        }

        if num1 < num2 {
            return -1
        } else if num1 > num2 {
            return 1
        }
    }

    return 0
}
```

### 进度跟踪机制

```go
// 进度包装读取器
type ProgressReader struct {
    reader      io.Reader
    total       int64
    downloaded  int64
    onProgress  func(downloaded, total int64)
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
    n, err := pr.reader.Read(p)
    pr.downloaded += int64(n)
    if pr.onProgress != nil {
        pr.onProgress(pr.downloaded, pr.total)
    }
    return n, err
}
```

## 🔄 版本管理

### 版本注入机制

**编译时版本注入**：
```bash
# 本地构建 (urlnav.go)
ldflags="-s -w -X main.Version=1.4.2 -X main.GitHubOwner=wangyaxings -X main.GitHubRepo=url-navigator"
wails build -ldflags "$ldflags"

# GitHub Actions构建
wails build -ldflags "-s -w -X main.Version=${{ steps.get_version.outputs.VERSION }}"
```

**配置文件版本**：
```json
// wails.json
{
  "info": {
    "version": "1.4.2"
  },
  "github": {
    "owner": "wangyaxings",
    "repo": "url-navigator"
  }
}
```

### 版本来源透明化

```go
func (a *App) GetCurrentVersionWithSource() map[string]interface{} {
    if RuntimeVersion != nil {
        return map[string]interface{}{
            "version":    ensureVersionPrefix(RuntimeVersion.Version),
            "source":     string(RuntimeVersion.Source),
            "is_default": RuntimeVersion.IsDefault,
            "reliable":   !RuntimeVersion.IsDefault,
        }
    }

    return map[string]interface{}{
        "version":    "v1.4.2-fallback",
        "source":     string(SourceDefault),
        "is_default": true,
        "reliable":   false,
    }
}
```

## ❗ 问题与解决方案

### 1. 版本显示"unknown"问题

**问题**：应用显示版本为"unknown"，用户无法知道真实版本

**根本原因**：
- 编译时未正确注入版本信息
- 配置文件版本读取失败
- 版本初始化逻辑缺陷

**解决方案**：
- ✅ 多级版本获取机制
- ✅ 版本来源透明化显示
- ✅ 兜底默认版本机制
- ✅ 调试信息提供

```go
// 修复前：可能返回unknown
func GetCurrentVersion() string {
    if version == "" {
        return "unknown"  // ❌ 用户困惑
    }
    return version
}

// 修复后：永远不返回unknown
func GetCurrentVersion() string {
    // 1. 尝试编译时版本
    if Version != "" && Version != "dev" {
        return ensureVersionPrefix(Version)
    }

    // 2. 尝试配置文件版本
    if version, err := readVersionFromConfig(); err == nil {
        return ensureVersionPrefix(version)
    }

    // 3. 兜底默认版本
    return "v1.4.2"  // ✅ 明确的版本
}
```

### 2. 本地与CI构建版本不一致

**问题**：本地构建显示正确版本，GitHub Actions构建显示"unknown"

**根本原因**：
- 本地构建脚本 (`urlnav.go`) 自动注入版本
- GitHub Actions缺少版本注入步骤

**解决方案**：
```yaml
# GitHub Actions 修复
- name: Extract version from wails.json
  id: get_version
  run: |
    $version = (Get-Content "wails.json" | ConvertFrom-Json).info.version
    echo "VERSION=$version" >> $env:GITHUB_OUTPUT

- name: Build with version injection
  run: |
    go run tools/urlnav.go build  # 使用统一构建脚本
```

### 3. 更新下载失败处理

**问题**：网络中断或服务器错误导致更新失败

**解决方案**：
- ✅ 超时控制和重试机制
- ✅ 详细的错误分类和提示
- ✅ 自动回滚机制
- ✅ 用户友好的错误消息

## 🎯 最佳实践

### 1. 版本管理最佳实践

**版本号格式**：
- 使用语义化版本 (Semantic Versioning)
- 格式: `vMAJOR.MINOR.PATCH`
- 示例: `v1.2.3`, `v2.0.0`, `v1.0.0-beta.1`

**版本注入策略**：
```bash
# 1. 编译时注入 (最高优先级)
-ldflags "-X main.Version=${VERSION}"

# 2. 配置文件备份
# wails.json, package.json, version.json

# 3. 环境变量支持
export APP_VERSION="1.2.3"

# 4. Git标签自动获取
VERSION=$(git describe --tags --abbrev=0)
```

### 2. 安全更新实践

**下载验证**：
```go
// SHA256校验
func verifyDownload(filePath, expectedHash string) error {
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()

    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return err
    }

    actualHash := hex.EncodeToString(hasher.Sum(nil))
    if actualHash != expectedHash {
        return fmt.Errorf("文件校验失败")
    }
    return nil
}
```

### 3. 用户体验最佳实践

**非阻塞更新**：
- 后台检查更新
- 用户确认后再下载
- 显示详细进度
- 支持取消操作

**错误处理**：
- 详细的错误分类
- 用户友好的错误消息
- 自动重试机制
- 降级方案

## 🚀 部署指南

### 1. GitHub Release 配置

**Release工作流**：
```yaml
# .github/workflows/release.yml
name: Release
on:
  push:
    tags: [ "v*.*.*" ]

jobs:
  build:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - name: Extract version
        id: version
        run: echo "VERSION=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT

      - name: Build with version injection
        run: go run tools/urlnav.go build

      - name: Create Release
        uses: ncipollo/release-action@v1
        with:
          artifacts: "build/bin/*.exe"
          generateReleaseNotes: true
```

### 2. 版本发布流程

**自动化发布脚本**：
```bash
#!/bin/bash
# release.sh

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "使用方法: ./release.sh v1.2.3"
    exit 1
fi

# 使用统一工具发布
go run tools/urlnav.go release $VERSION

echo "✅ 发布完成: $VERSION"
```

### 3. 监控和分析

**更新成功率监控**：
```go
type UpdateMetrics struct {
    CheckCount      int64     `json:"check_count"`
    UpdateCount     int64     `json:"update_count"`
    SuccessCount    int64     `json:"success_count"`
    FailureCount    int64     `json:"failure_count"`
    LastCheckTime   time.Time `json:"last_check_time"`
    LastUpdateTime  time.Time `json:"last_update_time"`
}

func (m *UpdateMetrics) RecordUpdate(success bool) {
    m.UpdateCount++
    if success {
        m.SuccessCount++
        m.LastUpdateTime = time.Now()
    } else {
        m.FailureCount++
    }
}
```

## 📚 总结

这个自动更新系统提供了一个完整的、生产就绪的解决方案，具有以下特点：

### 🎯 核心优势

1. **完全透明**：用户始终知道版本来源和可靠性
2. **健壮可靠**：多级错误处理和自动恢复机制
3. **用户友好**：直观的进度显示和错误提示
4. **易于集成**：模块化设计，可轻松移植到其他项目
5. **安全可靠**：文件校验、自动回滚

### 🔧 技术特色

- **多源版本管理**：编译时注入 + 配置文件 + 兜底机制
- **实时进度跟踪**：精确的下载进度和速度显示
- **平台适配**：支持不同操作系统的更新机制
- **错误恢复**：详细的错误分类和自动重试机制

### 🚀 未来扩展

- [ ] 增量更新支持
- [ ] 多平台并行构建
- [ ] 自定义更新服务器
- [ ] 离线更新包支持
- [ ] 更新回滚UI界面

这个文档提供了从设计理念到具体实现，从问题解决到最佳实践的完整指南，可以作为其他项目实现自动更新功能的参考模板和最佳实践指南。