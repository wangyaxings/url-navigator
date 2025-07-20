// version.go - 改进版本管理逻辑
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version     string `json:"version"`
	GitHubOwner string `json:"github_owner"`
	GitHubRepo  string `json:"github_repo"`
	AppName     string `json:"app_name"`
}

var (
	// 运行时版本信息，从配置文件加载
	RuntimeVersion *VersionInfo

	// 编译时注入的版本信息（通过 ldflags）
	Version     = "dev"           // -ldflags "-X main.Version=1.2.0"
	GitHubOwner = ""              // -ldflags "-X main.GitHubOwner=yourusername"
	GitHubRepo  = ""              // -ldflags "-X main.GitHubRepo=url-navigator"
	AppName     = "URLNavigator"
)

// GetCurrentVersion 获取当前应用版本
func (a *App) GetCurrentVersion() string {
	// 修复：确保总是返回实际的当前版本，而不是固定值
	if RuntimeVersion == nil {
		// 如果运行时版本未初始化，尝试重新初始化
		if err := InitVersionInfo(a.GetDataDir()); err != nil {
			// 初始化失败时，检查编译时注入的版本
			if Version != "dev" && Version != "" {
				return ensureVersionPrefix(Version)
			}
			// 作为最后的兜底，直接尝试从wails.json读取
			if version, _, _, err := readVersionFromWailsConfig(); err == nil && version != "" {
				return ensureVersionPrefix(version)
			}
			return "unknown"
		}
	}

	if RuntimeVersion.Version == "" || RuntimeVersion.Version == "unknown" {
		// 如果运行时版本为空，尝试使用编译时版本
		if Version != "dev" && Version != "" {
			return ensureVersionPrefix(Version)
		}
		// 作为兜底，直接从wails.json读取
		if version, _, _, err := readVersionFromWailsConfig(); err == nil && version != "" {
			return ensureVersionPrefix(version)
		}
		return "unknown"
	}

	return ensureVersionPrefix(RuntimeVersion.Version)
}

// ensureVersionPrefix 确保版本号有v前缀
func ensureVersionPrefix(version string) string {
	if version == "" || version == "unknown" || version == "dev" {
		return version
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// InitVersionInfo 初始化版本信息 - 改进的版本
func InitVersionInfo(dataDir string) error {
	// 1. 首先尝试从编译时注入的信息获取（最高优先级）
	if Version != "dev" && Version != "" {
		RuntimeVersion = &VersionInfo{
			Version:     Version,
			GitHubOwner: getGitHubOwner(),
			GitHubRepo:  getGitHubRepo(),
			AppName:     AppName,
		}

		return nil
	}

	// 2. 尝试从 wails.json 读取版本信息（第二优先级）
	version, owner, repo, err := readVersionFromWailsConfig()
	if err == nil && version != "" {
		RuntimeVersion = &VersionInfo{
			Version:     version,
			GitHubOwner: owner,
			GitHubRepo:  repo,
			AppName:     AppName,
		}

		return nil
	}

	// 3. 跳过用户配置文件，因为它可能是过时的

	// 4. 最后从根目录的 version.json 读取（兜底方案）
	if data, err := os.ReadFile("version.json"); err == nil {
		var config struct {
			Version string `json:"version"`
			GitHub  struct {
				Owner string `json:"owner"`
				Repo  string `json:"repo"`
			} `json:"github"`
		}
		if json.Unmarshal(data, &config) == nil && config.Version != "" {
			RuntimeVersion = &VersionInfo{
				Version:     config.Version,
				GitHubOwner: config.GitHub.Owner,
				GitHubRepo:  config.GitHub.Repo,
				AppName:     AppName,
			}

			return nil
		}
	}

	// 5. 如果所有方法都失败，创建一个默认配置
	RuntimeVersion = &VersionInfo{
		Version:     "unknown",
		GitHubOwner: getGitHubOwner(),
		GitHubRepo:  getGitHubRepo(),
		AppName:     AppName,
	}


	return fmt.Errorf("无法确定应用版本，请检查版本配置文件")
}

// readVersionFromWailsConfig 从 wails.json 读取版本信息
func readVersionFromWailsConfig() (version, owner, repo string, err error) {
	data, err := os.ReadFile("wails.json")
	if err != nil {
		return "", "", "", err
	}

	var config struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
		// wails.json 中的 github 配置
		GitHub struct {
			Owner string `json:"owner"`
			Repo  string `json:"repo"`
		} `json:"github"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", "", "", err
	}

	// 如果wails.json中没有GitHub信息，使用默认值
	owner = config.GitHub.Owner
	if owner == "" {
		owner = getGitHubOwner()
	}

	repo = config.GitHub.Repo
	if repo == "" {
		repo = getGitHubRepo()
	}

	return config.Info.Version, owner, repo, nil
}

// saveVersionConfig 保存版本配置到文件
func saveVersionConfig(configPath string, config *VersionInfo) error {
	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// GetVersionInfo 获取完整版本信息
func (a *App) GetVersionInfo() VersionInfo {
	if RuntimeVersion == nil {
		return VersionInfo{
			Version:     "unknown",
			GitHubOwner: getGitHubOwner(),
			GitHubRepo:  getGitHubRepo(),
			AppName:     AppName,
		}
	}

	// 返回一个副本，确保版本号格式正确
	info := *RuntimeVersion
	if info.Version != "" && info.Version != "unknown" && !strings.HasPrefix(info.Version, "v") {
		info.Version = "v" + info.Version
	}

	return info
}

// UpdateVersionConfig 更新版本配置
func (a *App) UpdateVersionConfig(owner, repo string) error {
	if RuntimeVersion == nil {
		return fmt.Errorf("版本信息未初始化")
	}

	RuntimeVersion.GitHubOwner = owner
	RuntimeVersion.GitHubRepo = repo

	dataDir := a.GetDataDir()
	configPath := filepath.Join(dataDir, "version.json")
	return saveVersionConfig(configPath, RuntimeVersion)
}

// ForceReloadVersion 强制重新加载版本信息（用于测试或调试）
func (a *App) ForceReloadVersion() error {
	dataDir := a.GetDataDir()
	return InitVersionInfo(dataDir)
}

// GetVersionFromWails 直接从wails.json获取版本信息（调试用）
func (a *App) GetVersionFromWails() (string, error) {
	version, _, _, err := readVersionFromWailsConfig()
	if err != nil {
		return "", err
	}

	if version != "" && !strings.HasPrefix(version, "v") {
		version = "v" + version
	}

	return version, nil
}

// getGitHubOwner 获取GitHub用户名，优先使用编译时注入的值
func getGitHubOwner() string {
	if GitHubOwner != "" {
		return GitHubOwner
	}
	return "wangyaxings" // 默认值
}

// getGitHubRepo 获取GitHub仓库名，优先使用编译时注入的值
func getGitHubRepo() string {
	if GitHubRepo != "" {
		return GitHubRepo
	}
	return "url-navigator" // 默认值
}