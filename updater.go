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
	CurrentVersion = "1.0.0"
	// GitHub仓库信息 - 请在创建仓库后替换这些值
	GitHubOwner = "wangyaxings"  // 替换为您的GitHub用户名
	GitHubRepo  = "url-navigator"         // 替换为您的仓库名
)

// CheckForUpdates checks if there's a new version available using real GitHub API
func (a *App) CheckForUpdates() UpdateInfo {
	currentVersion := CurrentVersion

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
			errorMsg = "API访问限制"
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
		Body    string `json:"body"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
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
		// 查找适合当前平台的下载链接
		expectedBinaryName := getBinaryName()
		for _, asset := range release.Assets {
			if asset.Name == expectedBinaryName {
				updateURL = asset.BrowserDownloadURL
				break
			}
		}

		// 如果没找到精确匹配，尝试模糊匹配
		if updateURL == "" {
			for _, asset := range release.Assets {
				if strings.Contains(asset.Name, runtime.GOOS) {
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

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 5 * time.Minute, // 5分钟超时，用于下载
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
		// 回滚失败的更新
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

// compareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func compareVersions(v1, v2 string) int {
	// 简单的版本比较实现
	// 对于生产环境，建议使用更完善的语义版本库

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
		num1, err1 := strconv.Atoi(parts1[i])
		num2, err2 := strconv.Atoi(parts2[i])

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

// getBinaryName returns the expected binary name for the current platform
func getBinaryName() string {
	switch runtime.GOOS {
	case "windows":
		return "URLNavigator.exe"
	case "darwin":
		return "URLNavigator.app"
	case "linux":
		return "URLNavigator"
	default:
		return "URLNavigator"
	}
}

// 测试方法已在app.go中定义，此处不再重复