package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Version     = "dev"      // -ldflags "-X main.Version=1.2.0"
	GitHubOwner = ""         // -ldflags "-X main.GitHubOwner=yourusername"
	GitHubRepo  = ""         // -ldflags "-X main.GitHubRepo=url-navigator"
	AppName     = "URLNavigator"
)

// InitVersionInfo 初始化版本信息
func InitVersionInfo(dataDir string) error {
	// 1. 首先尝试从编译时注入的信息获取
	if Version != "dev" && GitHubOwner != "" && GitHubRepo != "" {
		RuntimeVersion = &VersionInfo{
			Version:     Version,
			GitHubOwner: GitHubOwner,
			GitHubRepo:  GitHubRepo,
			AppName:     AppName,
		}
		return nil
	}

	// 2. 尝试从 wails.json 读取版本信息
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

	// 3. 尝试从用户配置文件读取
	configPath := filepath.Join(dataDir, "version.json")
	if data, err := os.ReadFile(configPath); err == nil {
		var config VersionInfo
		if json.Unmarshal(data, &config) == nil && config.Version != "" {
			RuntimeVersion = &config
			return nil
		}
	}

	// 4. 使用默认配置
	RuntimeVersion = &VersionInfo{
		Version:     "1.2.0",
		GitHubOwner: "", // 需要用户配置
		GitHubRepo:  "url-navigator",
		AppName:     AppName,
	}

	// 保存默认配置到文件
	return saveVersionConfig(configPath, RuntimeVersion)
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
		// 可以在 wails.json 中添加 github 配置
		GitHub struct {
			Owner string `json:"owner"`
			Repo  string `json:"repo"`
		} `json:"github"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return "", "", "", err
	}

	return config.Info.Version, config.GitHub.Owner, config.GitHub.Repo, nil
}

// saveVersionConfig 保存版本配置到文件
func saveVersionConfig(configPath string, config *VersionInfo) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// GetCurrentVersion 获取当前版本
func (a *App) GetCurrentVersion() string {
	if RuntimeVersion == nil {
		return "unknown"
	}
	return RuntimeVersion.Version
}

// GetVersionInfo 获取完整版本信息
func (a *App) GetVersionInfo() VersionInfo {
	if RuntimeVersion == nil {
		return VersionInfo{
			Version:     "unknown",
			GitHubOwner: "",
			GitHubRepo:  "",
			AppName:     AppName,
		}
	}
	return *RuntimeVersion
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