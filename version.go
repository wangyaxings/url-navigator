// version.go - 改进版本管理逻辑
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VersionSource 版本来源类型
type VersionSource string

const (
	SourceCompileTime VersionSource = "compile_time"   // 编译时注入
	SourceWailsJSON   VersionSource = "wails_json"     // wails.json文件
	SourceVersionJSON VersionSource = "version_json"   // version.json文件
	SourceDefault     VersionSource = "default"        // 默认值（兜底）
	SourceUnknown     VersionSource = "unknown"        // 无法确定
)

// VersionInfo 版本信息结构
type VersionInfo struct {
	Version     string        `json:"version"`
	GitHubOwner string        `json:"github_owner"`
	GitHubRepo  string        `json:"github_repo"`
	AppName     string        `json:"app_name"`
	Source      VersionSource `json:"source"`       // 版本来源
	IsDefault   bool          `json:"is_default"`   // 是否为默认值
}

var (
	// 运行时版本信息，从配置文件加载
	RuntimeVersion *VersionInfo

	// 编译时注入的版本信息（通过 ldflags）
	Version     = "1.4.2"        // -ldflags "-X main.Version=1.4.2" (设置默认值)
	GitHubOwner = "wangyaxings"   // -ldflags "-X main.GitHubOwner=wangyaxings"
	GitHubRepo  = "url-navigator" // -ldflags "-X main.GitHubRepo=url-navigator"
	AppName     = "URLNavigator"
)

// GetCurrentVersion 获取当前应用版本
func (a *App) GetCurrentVersion() string {
	// 确保版本信息已初始化
	if RuntimeVersion == nil {
		InitVersionInfo(a.GetDataDir())
	}

	if RuntimeVersion != nil {
		return ensureVersionPrefix(RuntimeVersion.Version)
	}

	// 如果仍然为nil，返回带标识的默认版本
	return "v1.4.2-default"
}

// GetCurrentVersionWithSource 获取当前版本及其来源信息
func (a *App) GetCurrentVersionWithSource() map[string]interface{} {
	// 确保版本信息已初始化
	if RuntimeVersion == nil {
		InitVersionInfo(a.GetDataDir())
	}

	if RuntimeVersion != nil {
		return map[string]interface{}{
			"version":    ensureVersionPrefix(RuntimeVersion.Version),
			"source":     string(RuntimeVersion.Source),
			"is_default": RuntimeVersion.IsDefault,
			"reliable":   !RuntimeVersion.IsDefault,
		}
	}

	// 完全失败的情况
	return map[string]interface{}{
		"version":    "v1.4.2-fallback",
		"source":     string(SourceDefault),
		"is_default": true,
		"reliable":   false,
	}
}

// ensureVersionPrefix 确保版本号有v前缀
func ensureVersionPrefix(version string) string {
	if version == "" || version == "unknown" || version == "dev" {
		// 如果是空值或unknown，返回默认版本
		if version == "" || version == "unknown" {
			return "v1.4.2"
		}
		return version
	}
	if !strings.HasPrefix(version, "v") {
		return "v" + version
	}
	return version
}

// InitVersionInfo 初始化版本信息 - 改进版本，记录版本来源
func InitVersionInfo(dataDir string) error {
	// 1. 首先尝试从编译时注入的信息获取（最高优先级）
	if Version != "" && Version != "dev" && Version != "unknown" && Version != "1.4.2" {
		RuntimeVersion = &VersionInfo{
			Version:     Version,
			GitHubOwner: getGitHubOwner(),
			GitHubRepo:  getGitHubRepo(),
			AppName:     AppName,
			Source:      SourceCompileTime,
			IsDefault:   false,
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
			Source:      SourceWailsJSON,
			IsDefault:   false,
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
				Source:      SourceVersionJSON,
				IsDefault:   false,
			}
			return nil
		}
	}

	// 5. 检查编译时是否设置了默认版本
	if Version == "1.4.2" {
		// 这可能是编译时设置的，但我们无法确定是真实版本还是我们的默认值
		RuntimeVersion = &VersionInfo{
			Version:     Version,
			GitHubOwner: getGitHubOwner(),
			GitHubRepo:  getGitHubRepo(),
			AppName:     AppName,
			Source:      SourceCompileTime,
			IsDefault:   true, // 标记为可能是默认值
		}
		return nil
	}

	// 6. 最后的兜底方案
	RuntimeVersion = &VersionInfo{
		Version:     "1.4.2",
		GitHubOwner: getGitHubOwner(),
		GitHubRepo:  getGitHubRepo(),
		AppName:     AppName,
		Source:      SourceDefault,
		IsDefault:   true,
	}

	return nil
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
	// 如果RuntimeVersion为nil，尝试初始化
	if RuntimeVersion == nil {
		if err := InitVersionInfo(a.GetDataDir()); err != nil {
			// 初始化失败，返回默认版本信息
			return VersionInfo{
				Version:     "1.4.2",
				GitHubOwner: getGitHubOwner(),
				GitHubRepo:  getGitHubRepo(),
				AppName:     AppName,
				Source:      SourceDefault,
				IsDefault:   true,
			}
		}
	}

	// 返回一个副本，确保版本号格式正确
	info := *RuntimeVersion
	if info.Version == "" || info.Version == "unknown" {
		info.Version = "1.4.2"
		info.Source = SourceDefault
		info.IsDefault = true
	}
	if !strings.HasPrefix(info.Version, "v") {
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

// DebugVersionInfo 获取版本调试信息，用于排查版本获取问题
func (a *App) DebugVersionInfo() map[string]interface{} {
	debug := make(map[string]interface{})

	// 编译时版本信息
	debug["compile_time_version"] = Version
	debug["compile_time_owner"] = GitHubOwner
	debug["compile_time_repo"] = GitHubRepo
	debug["app_name"] = AppName

	// 运行时版本信息
	if RuntimeVersion != nil {
		debug["runtime_version"] = RuntimeVersion.Version
		debug["runtime_owner"] = RuntimeVersion.GitHubOwner
		debug["runtime_repo"] = RuntimeVersion.GitHubRepo
		debug["runtime_app_name"] = RuntimeVersion.AppName
		debug["runtime_source"] = string(RuntimeVersion.Source)
		debug["runtime_is_default"] = RuntimeVersion.IsDefault
	} else {
		debug["runtime_version"] = "nil"
	}

	// 尝试从wails.json读取
	if version, owner, repo, err := readVersionFromWailsConfig(); err == nil {
		debug["wails_json_version"] = version
		debug["wails_json_owner"] = owner
		debug["wails_json_repo"] = repo
		debug["wails_json_available"] = true
	} else {
		debug["wails_json_error"] = err.Error()
		debug["wails_json_available"] = false
	}

	// 尝试从version.json读取
	if data, err := os.ReadFile("version.json"); err == nil {
		var config struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &config) == nil {
			debug["version_json_version"] = config.Version
			debug["version_json_available"] = true
		} else {
			debug["version_json_parse_error"] = "failed to parse"
			debug["version_json_available"] = false
		}
	} else {
		debug["version_json_error"] = err.Error()
		debug["version_json_available"] = false
	}

	// 当前获取到的版本和来源信息
	debug["current_version"] = a.GetCurrentVersion()
	versionWithSource := a.GetCurrentVersionWithSource()
	debug["current_version_source"] = versionWithSource["source"]
	debug["current_version_is_default"] = versionWithSource["is_default"]
	debug["current_version_reliable"] = versionWithSource["reliable"]

	// 数据目录
	debug["data_dir"] = a.GetDataDir()

	return debug
}