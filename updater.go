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
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

// CheckForUpdates 检查是否有新版本可用
func (a *App) CheckForUpdates() UpdateInfo {
	// 确保版本信息已初始化
	if RuntimeVersion == nil {
		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: "unknown",
			LatestVersion:  "unknown",
			UpdateURL:      "",
			ReleaseNotes:   "",
			ErrorMessage:   "版本信息未初始化，请检查配置",
		}
	}

	currentVersion := RuntimeVersion.Version

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

	// 检查GitHub仓库信息是否配置
	if RuntimeVersion.GitHubOwner == "" || RuntimeVersion.GitHubRepo == "" {
		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   "",
			ErrorMessage:   "GitHub仓库信息未配置，无法检查更新。请配置GitHub用户名和仓库名。",
		}
	}

	// 构建GitHub API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		RuntimeVersion.GitHubOwner, RuntimeVersion.GitHubRepo)

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
			ReleaseNotes:   "",
			ErrorMessage:   fmt.Sprintf("网络连接失败: %v", err),
		}
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		errorMsg := ""
		switch resp.StatusCode {
		case 404:
			errorMsg = fmt.Sprintf("仓库 %s/%s 未找到或没有发布版本",
				RuntimeVersion.GitHubOwner, RuntimeVersion.GitHubRepo)
		case 403:
			errorMsg = "GitHub API访问限制，请稍后重试"
		case 429:
			errorMsg = "请求过于频繁，请稍后重试"
		default:
			errorMsg = fmt.Sprintf("GitHub API请求失败，状态码: %d", resp.StatusCode)
		}

		return UpdateInfo{
			HasUpdate:      false,
			CurrentVersion: currentVersion,
			LatestVersion:  currentVersion,
			UpdateURL:      "",
			ReleaseNotes:   "",
			ErrorMessage:   errorMsg,
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
			ReleaseNotes:   "",
			ErrorMessage:   fmt.Sprintf("解析GitHub响应失败: %v", err),
		}
	}

	// 处理版本号（移除v前缀进行比较）
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	currentVersionForComparison := strings.TrimPrefix(currentVersion, "v")

	// 比较版本号
	hasUpdate := compareVersions(currentVersionForComparison, latestVersion) < 0

	var updateURL string
	if hasUpdate {
		// 查找Windows可执行文件
		updateURL = findWindowsExecutable(release.Assets)

		if updateURL == "" {
			return UpdateInfo{
				HasUpdate:      true,
				CurrentVersion: currentVersion,
				LatestVersion:  latestVersion,
				UpdateURL:      "",
				ReleaseNotes:   release.Body,
				ErrorMessage:   "新版本可用，但未找到Windows安装包",
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

// findWindowsExecutable 查找Windows可执行文件
func findWindowsExecutable(assets []struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size              int    `json:"size"`
}) string {
	// 优先查找确切的文件名
	for _, asset := range assets {
		if asset.Name == "URLNavigator.exe" {
			return asset.BrowserDownloadURL
		}
	}

	// 查找包含应用名称的.exe文件
	for _, asset := range assets {
		name := strings.ToLower(asset.Name)
		if strings.HasSuffix(name, ".exe") &&
		   (strings.Contains(name, "urlnavigator") || strings.Contains(name, "url-navigator")) {
			return asset.BrowserDownloadURL
		}
	}

	// 查找任何.exe文件
	for _, asset := range assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			return asset.BrowserDownloadURL
		}
	}

	return ""
}

// DownloadAndApplyUpdate 下载并应用更新
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

// compareVersions 比较两个版本号
// 返回: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
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

// TestUpdateAvailable 模拟有更新可用（用于测试）
func (a *App) TestUpdateAvailable() UpdateInfo {
	return UpdateInfo{
		HasUpdate:      true,
		CurrentVersion: a.GetCurrentVersion(),
		LatestVersion:  "999.999.999",
		UpdateURL:      "https://example.com/test-update.exe",
		ReleaseNotes:   "这是一个测试更新\n\n新功能:\n- 测试功能1\n- 测试功能2\n\n修复:\n- 修复了测试问题",
	}
}

// TestNoUpdate 模拟没有更新可用（用于测试）
func (a *App) TestNoUpdate() UpdateInfo {
	currentVersion := a.GetCurrentVersion()
	return UpdateInfo{
		HasUpdate:      false,
		CurrentVersion: currentVersion,
		LatestVersion:  currentVersion,
		UpdateURL:      "",
		ReleaseNotes:   "当前已是最新版本",
	}
}