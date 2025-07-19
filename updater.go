package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

// UpdateProgress represents download progress information
type UpdateProgress struct {
	Phase          string `json:"phase"`          // "downloading", "installing", "completed", "error"
	Progress       int    `json:"progress"`       // 0-100
	Speed          string `json:"speed"`          // Download speed (e.g. "1.2 MB/s")
	ETA            string `json:"eta"`            // Estimated time (e.g. "2m 30s")
	Downloaded     int64  `json:"downloaded"`     // Bytes downloaded
	Total          int64  `json:"total"`          // Total bytes
	Message        string `json:"message"`        // Status message
	Error          string `json:"error,omitempty"` // Error message if any
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

var (
	// 全局更新进度状态
	updateProgressMutex sync.RWMutex
	currentUpdateProgress *UpdateProgress
)

// GetUpdateProgress 获取当前更新进度
func (a *App) GetUpdateProgress() *UpdateProgress {
	updateProgressMutex.RLock()
	defer updateProgressMutex.RUnlock()

	if currentUpdateProgress == nil {
		return &UpdateProgress{
			Phase:    "idle",
			Progress: 0,
			Message:  "准备就绪",
		}
	}

	// 返回副本以避免并发问题
	progress := *currentUpdateProgress
	return &progress
}

// setUpdateProgress 设置更新进度
func setUpdateProgress(progress *UpdateProgress) {
	updateProgressMutex.Lock()
	defer updateProgressMutex.Unlock()
	currentUpdateProgress = progress
}

// ProgressReader wraps an io.Reader and provides progress tracking
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

// DownloadAndApplyUpdate 下载并应用更新
func (a *App) DownloadAndApplyUpdate(updateURL string) error {
	if updateURL == "" {
		return fmt.Errorf("无效的更新URL")
	}

	// 仅在Windows上支持自动更新
	if runtime.GOOS != "windows" {
		return fmt.Errorf("自动更新功能仅支持Windows版本")
	}

	// 初始化进度
	setUpdateProgress(&UpdateProgress{
		Phase:    "downloading",
		Progress: 0,
		Message:  "正在准备下载...",
	})

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Minute, // 30分钟超时，用于大文件下载
	}

	// 发起下载请求
	resp, err := client.Get(updateURL)
	if err != nil {
		setUpdateProgress(&UpdateProgress{
			Phase:   "error",
			Message: "下载失败",
			Error:   fmt.Sprintf("网络连接失败: %v", err),
		})
		return fmt.Errorf("下载更新失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		setUpdateProgress(&UpdateProgress{
			Phase:   "error",
			Message: "下载失败",
			Error:   fmt.Sprintf("HTTP状态码: %d", resp.StatusCode),
		})
		return fmt.Errorf("下载失败，状态码: %d", resp.StatusCode)
	}

	// 获取文件大小
	contentLength := resp.ContentLength
	if contentLength <= 0 {
		contentLength = 10 * 1024 * 1024 // 默认10MB，如果服务器没有提供文件大小
	}

	setUpdateProgress(&UpdateProgress{
		Phase:      "downloading",
		Progress:   0,
		Downloaded: 0,
		Total:      contentLength,
		Message:    "正在下载更新...",
	})

	// 创建进度追踪器
	startTime := time.Now()
	progressReader := &ProgressReader{
		reader: resp.Body,
		total:  contentLength,
		onProgress: func(downloaded, total int64) {
			elapsed := time.Since(startTime)

			// 计算进度百分比
			progress := int((downloaded * 100) / total)
			if progress > 100 {
				progress = 100
			}

			// 计算下载速度
			speed := ""
			if elapsed.Seconds() > 0 {
				bytesPerSecond := float64(downloaded) / elapsed.Seconds()
				speed = formatBytes(int64(bytesPerSecond)) + "/s"
			}

			// 计算剩余时间
			eta := ""
			if downloaded > 0 && elapsed.Seconds() > 1 {
				remainingBytes := total - downloaded
				bytesPerSecond := float64(downloaded) / elapsed.Seconds()
				if bytesPerSecond > 0 {
					remainingSeconds := float64(remainingBytes) / bytesPerSecond
					eta = formatDuration(time.Duration(remainingSeconds) * time.Second)
				}
			}

			setUpdateProgress(&UpdateProgress{
				Phase:      "downloading",
				Progress:   progress,
				Speed:      speed,
				ETA:        eta,
				Downloaded: downloaded,
				Total:      total,
				Message:    fmt.Sprintf("已下载 %s / %s", formatBytes(downloaded), formatBytes(total)),
			})
		},
	}

	// 开始安装阶段
	setUpdateProgress(&UpdateProgress{
		Phase:    "installing",
		Progress: 90,
		Message:  "正在安装更新...",
	})

	// 使用selfupdate进行更新
	err = selfupdate.Apply(progressReader, selfupdate.Options{})
	if err != nil {
		setUpdateProgress(&UpdateProgress{
			Phase:   "error",
			Message: "安装失败",
			Error:   fmt.Sprintf("更新失败: %v", err),
		})

		// 尝试回滚失败的更新
		if rollbackErr := selfupdate.RollbackError(err); rollbackErr != nil {
			return fmt.Errorf("更新失败且回滚失败: %v, 回滚错误: %v", err, rollbackErr)
		}
		return fmt.Errorf("更新失败: %v", err)
	}

	// 更新完成
	setUpdateProgress(&UpdateProgress{
		Phase:    "completed",
		Progress: 100,
		Message:  "更新完成，准备重启应用...",
	})

	// 延迟重启，给前端时间显示完成消息
	go func() {
		time.Sleep(2 * time.Second)
		a.RestartApplication()
	}()

	return nil
}

// RestartApplication 重启应用程序
func (a *App) RestartApplication() error {
	// 获取当前可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("无法获取可执行文件路径: %v", err)
	}

	// 在Windows上重启应用
	if runtime.GOOS == "windows" {
		// 使用 cmd /c start 来启动新实例并退出当前实例
		cmd := exec.Command("cmd", "/c", "start", "/b", exePath)
		cmd.Dir = filepath.Dir(exePath)

		// 启动新实例
		err := cmd.Start()
		if err != nil {
			return fmt.Errorf("重启失败: %v", err)
		}

		// 退出当前实例
		go func() {
			time.Sleep(500 * time.Millisecond) // 给新实例启动时间
			os.Exit(0)
		}()

		return nil
	}

	return fmt.Errorf("当前平台不支持自动重启")
}

// formatBytes 格式化字节数为人类可读格式
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration 格式化时间段为人类可读格式
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
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