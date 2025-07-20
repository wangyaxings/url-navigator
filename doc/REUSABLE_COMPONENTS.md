# 自动更新可重用组件包

> 本文档详细说明如何将自动更新功能抽象为可重用的Go包和React组件库，以便在其他项目中快速集成。

## 📦 包设计理念

### 设计目标

1. **模块化设计**：各组件独立，可按需使用
2. **零侵入集成**：最小化对现有项目的修改
3. **配置驱动**：通过配置文件控制所有行为
4. **类型安全**：完整的TypeScript支持
5. **生产就绪**：经过实际项目验证

### 架构概览

```
自动更新生态系统
├── 🔧 Go后端包 (go-autoupdate)
│   ├── 版本管理模块
│   ├── 更新检查模块
│   ├── 下载安装模块
│   └── 配置管理模块
├── 🎨 React前端包 (@org/react-autoupdate)
│   ├── Hooks库
│   ├── UI组件库
│   ├── 类型定义
│   └── 工具函数
└── 📚 集成示例
    ├── Wails项目示例
    ├── Electron项目示例
    └── 通用桌面应用示例
```

## 🔧 Go 后端包设计

### 包结构

```
github.com/yourorg/go-autoupdate/
├── autoupdate.go          # 主要API
├── config/
│   ├── config.go          # 配置管理
│   ├── validation.go      # 配置验证
│   └── loader.go          # 配置加载器
├── version/
│   ├── manager.go         # 版本管理器
│   ├── sources.go         # 版本来源处理
│   ├── comparison.go      # 版本比较算法
│   └── types.go           # 版本相关类型
├── updater/
│   ├── github.go          # GitHub API集成
│   ├── downloader.go      # 下载管理器
│   ├── progress.go        # 进度跟踪
│   ├── installer.go       # 安装逻辑
│   └── rollback.go        # 回滚机制
├── examples/
│   ├── basic/             # 基础使用示例
│   ├── wails/             # Wails集成示例
│   └── advanced/          # 高级配置示例
├── go.mod
└── README.md
```

### 主要API设计

```go
// autoupdate.go
package autoupdate

type AutoUpdater struct {
    config    *Config
    version   *version.Manager
    updater   *updater.Updater
}

// 创建自动更新器
func New(cfg *Config) (*AutoUpdater, error) {
    return &AutoUpdater{
        config:  cfg,
        version: version.NewManager(cfg.VersionConfig),
        updater: updater.New(cfg.UpdaterConfig),
    }, nil
}

// 检查更新
func (au *AutoUpdater) CheckForUpdates() (*UpdateInfo, error) {
    currentVersion := au.version.GetCurrentVersion()
    return au.updater.CheckUpdates(currentVersion)
}

// 下载并应用更新
func (au *AutoUpdater) DownloadAndApplyUpdate(updateURL string, progressCallback func(*UpdateProgress)) error {
    return au.updater.DownloadAndApply(updateURL, progressCallback)
}
```

### 配置结构

```go
// config/config.go
type Config struct {
    App        AppConfig        `json:"app"`
    GitHub     GitHubConfig     `json:"github"`
    Version    VersionConfig    `json:"version"`
    Update     UpdateConfig     `json:"update"`
}

type AppConfig struct {
    Name        string `json:"name"`
    DisplayName string `json:"display_name"`
    Version     string `json:"version"`
}

type GitHubConfig struct {
    Owner string `json:"owner"`
    Repo  string `json:"repo"`
    Token string `json:"token,omitempty"` // 可选的访问令牌
}

type VersionConfig struct {
    Sources    []string `json:"sources"`    // 版本来源优先级
    DefaultVal string   `json:"default"`    // 默认版本
}

type UpdateConfig struct {
    CheckInterval   time.Duration `json:"check_interval"`   // 检查间隔
    DownloadTimeout time.Duration `json:"download_timeout"` // 下载超时
    RetryCount      int          `json:"retry_count"`      // 重试次数
    AutoInstall     bool         `json:"auto_install"`     // 自动安装
}
```

## 🎨 React 前端包设计

### 包结构

```
@yourorg/react-autoupdate/
├── src/
│   ├── index.ts           # 主要导出
│   ├── hooks/
│   │   ├── useAutoUpdate.ts      # 自动更新Hook
│   │   ├── useVersionInfo.ts     # 版本信息Hook
│   │   └── useUpdateProgress.ts  # 进度跟踪Hook
│   ├── components/
│   │   ├── UpdateButton.tsx      # 更新按钮
│   │   ├── UpdateDialog.tsx      # 更新对话框
│   │   ├── ProgressDialog.tsx    # 进度对话框
│   │   ├── VersionDisplay.tsx    # 版本显示
│   │   └── UpdateNotification.tsx # 更新通知
│   ├── types/
│   │   ├── update.ts      # 更新相关类型
│   │   └── version.ts     # 版本相关类型
│   ├── utils/
│   │   ├── format.ts      # 格式化工具
│   │   └── validation.ts  # 验证工具
│   └── styles/
│       └── components.css # 默认样式
├── examples/
├── package.json
└── README.md
```

### 核心Hook实现

```typescript
// hooks/useAutoUpdate.ts
export interface UseAutoUpdateOptions {
    checkOnMount?: boolean;
    checkInterval?: number;
    onUpdateAvailable?: (info: UpdateInfo) => void;
    onUpdateComplete?: () => void;
    onError?: (error: string) => void;
}

export function useAutoUpdate(options: UseAutoUpdateOptions = {}) {
    const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
    const [isChecking, setIsChecking] = useState(false);
    const [isUpdating, setIsUpdating] = useState(false);
    const [progress, setProgress] = useState<UpdateProgress | null>(null);

    const checkForUpdates = useCallback(async () => {
        setIsChecking(true);
        try {
            const info = await window.autoupdate.checkForUpdates();
            setUpdateInfo(info);
            if (info.hasUpdate && options.onUpdateAvailable) {
                options.onUpdateAvailable(info);
            }
        } catch (error) {
            options.onError?.(error.message);
        } finally {
            setIsChecking(false);
        }
    }, [options]);

    const downloadAndApplyUpdate = useCallback(async (updateURL: string) => {
        setIsUpdating(true);
        try {
            await window.autoupdate.downloadAndApplyUpdate(updateURL);
            options.onUpdateComplete?.();
        } catch (error) {
            options.onError?.(error.message);
        } finally {
            setIsUpdating(false);
        }
    }, [options]);

    return {
        updateInfo,
        isChecking,
        isUpdating,
        progress,
        checkForUpdates,
        downloadAndApplyUpdate,
    };
}
```

### UI组件实现

```typescript
// components/UpdateButton.tsx
export interface UpdateButtonProps {
    onUpdateAvailable?: (info: UpdateInfo) => void;
    onUpdateComplete?: () => void;
    onError?: (error: string) => void;
    className?: string;
    children?: React.ReactNode;
}

export function UpdateButton({
    onUpdateAvailable,
    onUpdateComplete,
    onError,
    className,
    children = "检查更新"
}: UpdateButtonProps) {
    const { isChecking, checkForUpdates } = useAutoUpdate({
        onUpdateAvailable,
        onUpdateComplete,
        onError,
    });

    return (
        <button
            onClick={checkForUpdates}
            disabled={isChecking}
            className={`autoupdate-btn ${className || ''}`}
        >
            {isChecking ? (
                <>
                    <span className="autoupdate-spinner" />
                    检查中...
                </>
            ) : (
                children
            )}
        </button>
    );
}
```

```typescript
// components/UpdateDialog.tsx
export interface UpdateDialogProps {
    isOpen: boolean;
    updateInfo: UpdateInfo | null;
    onConfirm: () => void;
    onCancel: () => void;
    className?: string;
}

export function UpdateDialog({
    isOpen,
    updateInfo,
    onConfirm,
    onCancel,
    className = ''
}: UpdateDialogProps) {
    if (!isOpen || !updateInfo?.hasUpdate) {
        return null;
    }

    return (
        <div className={`autoupdate-dialog-overlay ${className}`}>
            <div className="autoupdate-dialog">
                <div className="autoupdate-dialog-header">
                    <h3>🚀 发现新版本</h3>
                    <button
                        className="autoupdate-dialog-close"
                        onClick={onCancel}
                        aria-label="关闭"
                    >
                        ✕
                    </button>
                </div>

                <div className="autoupdate-dialog-content">
                    <div className="autoupdate-version-info">
                        <div className="autoupdate-version-row">
                            <span>当前版本:</span>
                            <span className="autoupdate-version-current">
                                {updateInfo.currentVersion}
                            </span>
                        </div>
                        <div className="autoupdate-version-row">
                            <span>最新版本:</span>
                            <span className="autoupdate-version-latest">
                                {updateInfo.latestVersion}
                            </span>
                        </div>
                    </div>

                    {updateInfo.releaseNotes && (
                        <div className="autoupdate-release-notes">
                            <h4>更新内容:</h4>
                            <div className="autoupdate-release-content">
                                <pre>{updateInfo.releaseNotes}</pre>
                            </div>
                        </div>
                    )}
                </div>

                <div className="autoupdate-dialog-footer">
                    <button
                        className="autoupdate-btn autoupdate-btn-secondary"
                        onClick={onCancel}
                    >
                        稍后更新
                    </button>
                    <button
                        className="autoupdate-btn autoupdate-btn-primary"
                        onClick={onConfirm}
                    >
                        立即更新
                    </button>
                </div>
            </div>
        </div>
    );
}
```

## 📚 集成示例

### Go 后端集成

```go
// main.go
package main

import (
    "log"

    "github.com/yourorg/go-autoupdate"
    "github.com/yourorg/go-autoupdate/config"
)

func main() {
    // 创建配置
    cfg := &config.Config{
        App: config.AppConfig{
            Name:        "MyApp",
            DisplayName: "My Application",
            Version:     "1.0.0",
        },
        GitHub: config.GitHubConfig{
            Owner: "myorg",
            Repo:  "myapp",
        },
    }

    // 创建自动更新器
    updater, err := autoupdate.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer updater.Close()

    // 检查更新
    updateInfo, err := updater.CheckForUpdates()
    if err != nil {
        log.Printf("检查更新失败: %v", err)
        return
    }

    if updateInfo.HasUpdate {
        log.Printf("发现新版本: %s", updateInfo.LatestVersion)

        // 下载并应用更新
        err = updater.DownloadAndApplyUpdate(updateInfo.UpdateURL, func(progress *autoupdate.UpdateProgress) {
            log.Printf("更新进度: %d%% - %s", progress.Progress, progress.Message)
        })
        if err != nil {
            log.Printf("更新失败: %v", err)
        }
    }
}
```

### React 前端集成

```tsx
// App.tsx
import React from 'react';
import {
    UpdateButton,
    UpdateDialog,
    ProgressDialog,
    useAutoUpdate
} from '@yourorg/react-autoupdate';

function App() {
    const {
        updateInfo,
        isChecking,
        isUpdating,
        progress,
        checkForUpdates,
        downloadAndApplyUpdate,
    } = useAutoUpdate({
        checkOnMount: true,
        checkInterval: 24 * 60 * 60 * 1000, // 24小时
        onUpdateAvailable: (info) => {
            console.log('发现新版本:', info.latestVersion);
            setShowUpdateDialog(true);
        },
        onUpdateComplete: () => {
            console.log('更新完成');
        },
        onError: (error) => {
            console.error('更新错误:', error);
        },
    });

    const [showUpdateDialog, setShowUpdateDialog] = React.useState(false);
    const [showProgressDialog, setShowProgressDialog] = React.useState(false);

    const handleUpdateConfirm = async () => {
        if (updateInfo?.updateUrl) {
            setShowProgressDialog(true);
            setShowUpdateDialog(false);
            await downloadAndApplyUpdate(updateInfo.updateUrl);
        }
    };

    return (
        <div className="app">
            <header>
                <h1>My Application</h1>
                <UpdateButton
                    onUpdateAvailable={() => setShowUpdateDialog(true)}
                />
            </header>

            <main>
                {/* 应用主要内容 */}
            </main>

            <UpdateDialog
                isOpen={showUpdateDialog}
                updateInfo={updateInfo}
                onConfirm={handleUpdateConfirm}
                onCancel={() => setShowUpdateDialog(false)}
            />

            <ProgressDialog
                isOpen={showProgressDialog}
                progress={progress}
            />
        </div>
    );
}

export default App;
```

### Wails项目集成

```go
// wails-app/app.go
package main

import (
    "context"

    "github.com/yourorg/go-autoupdate"
)

type App struct {
    ctx      context.Context
    updater  *autoupdate.AutoUpdater
}

func NewApp() *App {
    return &App{}
}

func (a *App) OnStartup(ctx context.Context) {
    a.ctx = ctx

    // 初始化自动更新器
    cfg := &autoupdate.Config{
        App: autoupdate.AppConfig{
            Name: "MyWailsApp",
            Version: "1.0.0",
        },
        GitHub: autoupdate.GitHubConfig{
            Owner: "myorg",
            Repo: "my-wails-app",
        },
    }

    updater, err := autoupdate.New(cfg)
    if err != nil {
        fmt.Printf("初始化更新器失败: %v", err)
        return
    }

    a.updater = updater
}

// 暴露给前端的方法
func (a *App) CheckForUpdates() (*autoupdate.UpdateInfo, error) {
    return a.updater.CheckForUpdates()
}

func (a *App) DownloadAndApplyUpdate(updateURL string) error {
    return a.updater.DownloadAndApplyUpdate(updateURL, nil)
}

func (a *App) GetUpdateProgress() (*autoupdate.UpdateProgress, error) {
    return a.updater.GetUpdateProgress()
}
```

## 🎯 配置文件示例

### autoupdate.json

```json
{
  "app": {
    "name": "MyApp",
    "display_name": "My Application",
    "version": "1.0.0",
    "executable": "MyApp.exe"
  },
  "github": {
    "owner": "myorg",
    "repo": "myapp",
    "token": "",
    "asset_pattern": "*.exe"
  },
  "version": {
    "sources": ["compile_time", "wails_json", "version_json", "default"],
    "default": "1.0.0",
    "prefix": "v"
  },
  "update": {
    "check_interval": "24h",
    "download_timeout": "30m",
    "retry_count": 3,
    "auto_install": false,
    "backup_count": 3
  }
}
```

## 🚀 发布和分发

### NPM包发布

```bash
# 构建React包
cd react-autoupdate
npm run build

# 发布到NPM
npm publish --access public
```

### Go模块发布

```bash
# 打标签
git tag v1.0.0

# 推送标签
git push origin v1.0.0

# Go模块自动可用
go get github.com/yourorg/go-autoupdate@v1.0.0
```

### 文档网站

```bash
# 使用VitePress生成文档
npm create vitepress docs
cd docs
npm run build
npm run deploy
```

## 📈 使用统计

### 包使用指标

```go
// 统计使用情况
type UsageMetrics struct {
    PackageVersion    string    `json:"package_version"`
    InstallCount      int64     `json:"install_count"`
    ActiveUsers       int64     `json:"active_users"`
    SuccessRate       float64   `json:"success_rate"`
    LastReported      time.Time `json:"last_reported"`
}
```

### 反馈收集

```typescript
// 收集用户反馈
interface Feedback {
    version: string;
    platform: string;
    success: boolean;
    error?: string;
    duration: number;
    timestamp: Date;
}
```

## 🔮 未来规划

### 短期目标 (3个月)

- [ ] 完成Go包和React包的基础版本
- [ ] 提供完整的文档和示例
- [ ] 支持Windows、macOS、Linux三个平台
- [ ] 发布到GitHub、NPM、Go模块仓库

### 中期目标 (6个月)

- [ ] 添加更多的版本源支持 (GitLab、自定义服务器)
- [ ] 增量更新支持
- [ ] 图形化配置工具
- [ ] VS Code扩展支持

### 长期目标 (1年)

- [ ] 多语言支持 (Rust、Python、C++)
- [ ] 云端配置管理
- [ ] 企业级功能 (签名验证、权限控制)
- [ ] 统计分析仪表板

## 📞 社区支持

### 贡献指南

1. **Fork项目**到自己的GitHub账户
2. **创建功能分支** `git checkout -b feature/amazing-feature`
3. **提交更改** `git commit -m 'Add amazing feature'`
4. **推送分支** `git push origin feature/amazing-feature`
5. **创建Pull Request**

### 问题反馈

- 🐛 **Bug报告**: [GitHub Issues](https://github.com/yourorg/go-autoupdate/issues)
- 💡 **功能请求**: [GitHub Discussions](https://github.com/yourorg/go-autoupdate/discussions)
- 📧 **商业支持**: support@yourorg.com

### 社区资源

- 📚 **文档网站**: https://autoupdate.yourorg.com
- 💬 **Discord社区**: https://discord.gg/autoupdate
- 🐦 **Twitter更新**: [@yourorg_dev](https://twitter.com/yourorg_dev)

## 📄 许可证

MIT License - 详见 [LICENSE](LICENSE) 文件

---

通过这个可重用组件包，任何桌面应用都可以在几分钟内添加专业级的自动更新功能！🚀