package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/minio/selfupdate"
)

// UpdateInfo represents update information
type UpdateInfo struct {
	HasUpdate      bool   `json:"hasUpdate"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	UpdateURL      string `json:"updateUrl"`
	ReleaseNotes   string `json:"releaseNotes"`
}

const (
	CurrentVersion = "1.2.0"
	// GitHub仓库信息 - 请替换为您的实际仓库信息
	GitHubOwner = "YOUR_GITHUB_USERNAME"  // 替换为您的GitHub用户名
	GitHubRepo  = "url-navigator"         // 替换为您的仓库名
)

// CheckForUpdates checks if there's a new version available using GitHub API
func (a *App) CheckForUpdates() UpdateInfo {
	currentVersion := CurrentVersion

	// 仅在Windows上支持自动更新
	if runtime.GOOS != "windows" {
		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   "自动更新功能仅支持Windows版本",
		}
	}

	// 构建GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", GitHubOwner, GitHubRepo)

	// 创建HTTP客户端，设置超时
	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	// 发送请求
	resp, err := client.Get(apiURL)
	if err != nil {
		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   fmt.Sprintf("网络连接失败: %v", err),
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		errorMsg := ""
		switch resp.StatusCode {
		case 404:
			errorMsg = "仓库未找到或没有发布版本"
		case 403:
			errorMsg = "API访问限制，请稍后重试"
		case 429:
			errorMsg = "请求过于频繁，请稍后重试"
		default:
			errorMsg = fmt.Sprintf("API请求失败，状态码: %d", resp.StatusCode)
		}

		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   errorMsg,
		}
	}

	// 解析GitHub API响应
	var release struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		Body    string `json:"body"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size              int    `json:"size"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   fmt.Sprintf("解析更新信息失败: %v", err),
		}
	}

	// 处理版本号（移除v前缀）
	latestVersion := strings.TrimPrefix(release.TagName, "v")

	// 比较版本号
	hasUpdate := compareVersions(currentVersion, latestVersion) < 0

	var updateURL string
	if hasUpdate {
		// 查找Windows可执行文件
		for _, asset := range release.Assets {
			// 查找 URLNavigator.exe 文件
			if asset.Name == "URLNavigator.exe" {
				updateURL = asset.BrowserDownloadURL
				break
			}
		}

		// 如果没找到确切的文件名，尝试查找包含.exe的文件
		if updateURL == "" {
			for _, asset := range release.Assets {
				if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") &&
				   strings.Contains(strings.ToLower(asset.Name), "urlnavigator") {
					updateURL = asset.BrowserDownloadURL
					break
				}
			}
		}
	}

	return UpdateInfo{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		UpdateURL:      updateURL,
		ReleaseNotes:   release.Body,
	}
}

// DownloadAndApplyUpdate downloads and applies an update using selfupdate
func (a *App) DownloadAndApplyUpdate(updateURL string) error {
	if updateURL == "" {
		return fmt.Errorf("无效的更新URL")
	}

	// 仅在Windows上支持自动更新
	if runtime.GOOS != "windows" {
		return fmt.Errorf("自动更新功能仅支持Windows版本")
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Minute, // 10分钟超时，用于下载
	}

	// 下载新版本
	resp, err := client.Get(updateURL)
	if err != nil {
		return fmt.Errorf("下载更新失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 使用selfupdate进行更新
	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		// 尝试回滚失败的更新
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("更新失败且回滚失败: %v, 回滚错误: %v", err, rollbackErr)
		}
		return fmt.Errorf("更新失败: %v", err)
	}

	return nil
}

// GetCurrentVersion returns the current version
func (a *App) GetCurrentVersion() string {
	return CurrentVersion
}

// compareVersions compares two version strings using semantic versioning
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
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
		// 提取数字部分（忽略预发布标识符）
		num1Str := strings.Split(parts1[i], "-")[0]
		num2Str := strings.Split(parts2[i], "-")[0]

		num1, err1 := strconv.Atoi(num1Str)
		num2, err2 := strconv.Atoi(num2Str)

		if err1 != nil || err2 != nil {
			// 如果无法解析为数字，进行字符串比较
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

// TestUpdateAvailable simulates an update being available (for testing)
func (a *App) TestUpdateAvailable() UpdateInfo {
	return UpdateInfo{
		HasUpdate:      true,
		CurrentVersion: CurrentVersion,
		LatestVersion:  "999.999.999",
		UpdateURL:      "https://example.com/test-update.exe",
		ReleaseNotes:   "这是一个测试更新\n\n新功能:\n- 测试功能1\n- 测试功能2\n\n修复:\n- 修复了测试问题",
	}
}

// TestNoUpdate simulates no update being available (for testing)
func (a *App) TestNoUpdate() UpdateInfo {
	return UpdateInfo{
		HasUpdate:      false,
		CurrentVersion: CurrentVersion,
		LatestVersion:  CurrentVersion,
		UpdateURL:      "",
		ReleaseNotes:   "当前已是最新版本",
	}
}