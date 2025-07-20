package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// 配置结构体
type Config struct {
	Version string `json:"version"`
	GitHub  struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	} `json:"github"`
	App struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	} `json:"app"`
	Build struct {
		Platform string   `json:"platform"`
		Flags    []string `json:"flags"`
		LdFlags  []string `json:"ldflags"`
	} `json:"build"`
	Release struct {
		CreateGithubRelease   bool   `json:"create_github_release"`
		AutoOpenBrowser       bool   `json:"auto_open_browser"`
		CommitMessageTemplate string `json:"commit_message_template"`
		TagMessageTemplate    string `json:"tag_message_template"`
	} `json:"release"`
}

type WailsConfig struct {
	Info struct {
		Version string `json:"version"`
	} `json:"info"`
	GitHub struct {
		Owner string `json:"owner"`
		Repo  string `json:"repo"`
	} `json:"github"`
}

type PackageConfig struct {
	Version string `json:"version"`
}

type ReleaseOptions struct {
	Version     string
	SkipBuild   bool
	SkipRelease bool
	Force       bool
}

type GitRetryConfig struct {
	CodePushRetries int
	CodePushDelay   int
	TagPushRetries  int
	TagPushDelay    int
}

// 颜色输出函数
func colorPrint(color, prefix, message string) {
	colors := map[string]string{
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"reset":   "\033[0m",
	}

	fmt.Printf("%s[%s]%s %s\n", colors[color], prefix, colors["reset"], message)
}

func writeInfo(message string) {
	colorPrint("cyan", "INFO", message)
}

func writeSuccess(message string) {
	colorPrint("green", "SUCCESS", message)
}

func writeWarning(message string) {
	colorPrint("yellow", "WARNING", message)
}

func writeError(message string) {
	colorPrint("red", "ERROR", message)
}

func writeHeader(message string) {
	fmt.Println()
	fmt.Println("==========================================")
	colorPrint("magenta", "", message)
	fmt.Println("==========================================")
}

// 工具检查函数
func checkPrerequisites() bool {
	writeInfo("检查前置条件...")

	tools := map[string]string{
		"git":   "Git",
		"go":    "Go",
		"wails": "Wails",
	}

	allGood := true
	for tool, name := range tools {
		if err := exec.Command(tool, "version").Run(); err != nil {
			writeError(fmt.Sprintf("❌ %s 未安装或不在PATH中", name))
			allGood = false
		} else {
			writeSuccess(fmt.Sprintf("✅ %s 可用", name))
		}
	}

	// 检查Yarn（在frontend目录中）
	if _, err := os.Stat("frontend"); err == nil {
		if err := exec.Command("yarn", "--version").Run(); err != nil {
			writeError("❌ Yarn 未安装或不在PATH中")
			allGood = false
		} else {
			writeSuccess("✅ Yarn 可用")
		}
	}

	return allGood
}

// 版本格式验证
func validateVersionFormat(version string) (string, bool) {
	// 支持 vX.Y.Z 或 X.Y.Z 格式
	versionRegex := regexp.MustCompile(`^v?(\d+\.\d+\.\d+)$`)
	matches := versionRegex.FindStringSubmatch(version)
	if matches == nil {
		writeError("版本格式无效。请使用 vX.Y.Z 或 X.Y.Z 格式")
		return "", false
	}

	cleanVersion := matches[1] // 不带v前缀的版本号
	return cleanVersion, true
}

// 检查Git仓库状态
func checkGitRepository() bool {
	writeInfo("检查Git仓库状态...")

	// 检查是否在Git仓库中
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		writeError("当前目录不是Git仓库")
		return false
	}

	// 检查工作目录是否干净
	cmd := exec.Command("git", "status", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		writeError("无法检查Git状态")
		return false
	}

	if len(strings.TrimSpace(string(output))) > 0 {
		writeError("工作目录有未提交的更改。请先提交或暂存更改：")
		fmt.Println(string(output))
		return false
	}

	writeSuccess("✅ Git仓库状态正常")
	return true
}

// 加载配置
func loadConfig() (*Config, error) {
	writeInfo("加载配置文件...")

	// 检查必需文件
	requiredFiles := []string{"version.json", "wails.json"}
	for _, file := range requiredFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, fmt.Errorf("必需文件不存在: %s", file)
		}
	}

	// 读取version.json
	data, err := os.ReadFile("version.json")
	if err != nil {
		return nil, fmt.Errorf("读取version.json失败: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析version.json失败: %v", err)
	}

	writeSuccess("配置加载成功")
	return &config, nil
}

// 获取当前版本
func getCurrentVersion() (string, error) {
	writeInfo("检测当前版本...")

	// 首先尝试从wails.json读取
	if data, err := os.ReadFile("wails.json"); err == nil {
		var wailsConfig WailsConfig
		if err := json.Unmarshal(data, &wailsConfig); err == nil {
			if wailsConfig.Info.Version != "" {
				writeSuccess(fmt.Sprintf("从wails.json检测到当前版本: v%s", wailsConfig.Info.Version))
				return wailsConfig.Info.Version, nil
			}
		}
	}

	// 尝试从frontend/package.json读取
	if data, err := os.ReadFile("frontend/package.json"); err == nil {
		var packageConfig PackageConfig
		if err := json.Unmarshal(data, &packageConfig); err == nil {
			if packageConfig.Version != "" {
				writeSuccess(fmt.Sprintf("从frontend/package.json检测到当前版本: v%s", packageConfig.Version))
				return packageConfig.Version, nil
			}
		}
	}

	return "", fmt.Errorf("无法检测当前版本")
}

// 获取GitHub信息
func getGitHubInfo(config *Config) (string, string, error) {
	writeInfo("检测GitHub仓库信息...")

	// 尝试从git remote自动检测
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.Output()
	if err == nil {
		remoteUrl := strings.TrimSpace(string(output))
		githubRegex := regexp.MustCompile(`github\.com[:/]([^/]+)/([^/\.]+)`)
		matches := githubRegex.FindStringSubmatch(remoteUrl)
		if len(matches) >= 3 {
			owner := matches[1]
			repo := matches[2]
			writeSuccess(fmt.Sprintf("自动检测到GitHub仓库: %s/%s", owner, repo))
			return owner, repo, nil
		}
	}

	// 使用配置文件中的值
	owner := config.GitHub.Owner
	repo := config.GitHub.Repo
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("无法确定GitHub仓库信息")
	}

	writeInfo(fmt.Sprintf("使用配置文件中的GitHub仓库: %s/%s", owner, repo))
	return owner, repo, nil
}

// 确认操作
func confirmOperation(currentVersion, newVersion string, opts ReleaseOptions) bool {
	if opts.Force {
		writeInfo("强制模式已启用，跳过确认")
		return true
	}

	fmt.Println()
	fmt.Println("将执行以下操作:")
	fmt.Printf("- 更新版本从 v%s 到 v%s\n", currentVersion, newVersion)

	if !opts.SkipBuild {
		fmt.Println("- 构建Windows应用程序")
	}

	if !opts.SkipRelease {
		fmt.Println("- 创建Git标签并推送到仓库")
		fmt.Println("- 触发GitHub Actions构建")
	}

	fmt.Print("\n确认继续? (y/N): ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return response == "y" || response == "yes"
}

// 更新版本文件
func updateVersionFiles(newVersion, owner, repo string) error {
	writeInfo("更新版本文件...")
	writeInfo("注意: version.json 将保留为配置模板")

	// 更新wails.json
	if err := updateWailsConfig(newVersion, owner, repo); err != nil {
		return fmt.Errorf("更新wails.json失败: %v", err)
	}

	// 更新frontend/package.json
	if err := updatePackageConfig(newVersion); err != nil {
		writeWarning(fmt.Sprintf("更新frontend/package.json失败: %v", err))
	}

	writeSuccess("版本文件更新成功")
	return nil
}

func updateWailsConfig(newVersion, owner, repo string) error {
	data, err := os.ReadFile("wails.json")
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// 确保info部分存在
	if config["info"] == nil {
		config["info"] = make(map[string]interface{})
	}
	info := config["info"].(map[string]interface{})
	info["version"] = newVersion

	// 确保github部分存在并更新仓库信息
	if config["github"] == nil {
		config["github"] = make(map[string]interface{})
	}
	github := config["github"].(map[string]interface{})
	github["owner"] = owner
	github["repo"] = repo

	// 写入JSON文件
	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile("wails.json", updatedData, 0644); err != nil {
		return err
	}

	writeSuccess(fmt.Sprintf("更新wails.json到版本 %s", newVersion))
	return nil
}

func updatePackageConfig(newVersion string) error {
	packagePath := "frontend/package.json"
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return nil // 文件不存在，跳过
	}

	data, err := os.ReadFile(packagePath)
	if err != nil {
		return err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	config["version"] = newVersion

	updatedData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(packagePath, updatedData, 0644); err != nil {
		return err
	}

	writeSuccess(fmt.Sprintf("更新frontend/package.json到版本 %s", newVersion))
	return nil
}

// 构建应用程序
func buildApplication(newVersion, owner, repo string, config *Config) error {
	writeInfo("开始构建应用程序...")

	// 安装frontend依赖
	if _, err := os.Stat("frontend"); err == nil {
		writeInfo("安装frontend依赖...")
		cmd := exec.Command("yarn", "install", "--frozen-lockfile")
		cmd.Dir = "frontend"
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("yarn install失败: %v", err)
		}
		writeSuccess("Frontend依赖安装完成")

		// 构建frontend
		writeInfo("构建frontend...")
		cmd = exec.Command("yarn", "build")
		cmd.Dir = "frontend"
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("Frontend构建失败: %v", err)
		}
		writeSuccess("Frontend构建完成")
	}

	// 构建Wails应用程序
	writeInfo("构建Wails应用程序...")

	// 构建ldflags
	ldflags := append(config.Build.LdFlags,
		fmt.Sprintf("-X main.Version=%s", newVersion),
		fmt.Sprintf("-X main.GitHubOwner=%s", owner),
		fmt.Sprintf("-X main.GitHubRepo=%s", repo),
	)

	ldflagsString := strings.Join(ldflags, " ")

	// 构建wails build命令
	args := []string{
		"build",
		"-platform", config.Build.Platform,
		"-ldflags", ldflagsString,
	}
	args = append(args, config.Build.Flags...)

	writeInfo(fmt.Sprintf("构建命令: wails %s", strings.Join(args, " ")))

	cmd := exec.Command("wails", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Wails构建失败: %v", err)
	}

	writeSuccess("应用程序构建完成")
	return nil
}

// Git操作
func performGitOperations(newVersion string, config *Config) error {
	writeInfo("执行Git操作...")

	retryConfig := GitRetryConfig{
		CodePushRetries: 3,
		CodePushDelay:   5,
		TagPushRetries:  5,
		TagPushDelay:    8,
	}

	// 添加更改的文件
	filesToAdd := []string{"wails.json"}
	if _, err := os.Stat("frontend/package.json"); err == nil {
		filesToAdd = append(filesToAdd, "frontend/package.json")
	}

	for _, file := range filesToAdd {
		if err := exec.Command("git", "add", file).Run(); err != nil {
			return fmt.Errorf("添加文件%s失败: %v", file, err)
		}
	}

	// 提交更改
	commitMessage := strings.ReplaceAll(config.Release.CommitMessageTemplate, "{version}", "v"+newVersion)
	if err := exec.Command("git", "commit", "-m", commitMessage).Run(); err != nil {
		return fmt.Errorf("提交失败: %v", err)
	}
	writeSuccess("更改已提交")

	// 创建标签
	tagMessage := strings.ReplaceAll(config.Release.TagMessageTemplate, "{version}", "v"+newVersion)
	tagName := "v" + newVersion
	if err := exec.Command("git", "tag", "-a", tagName, "-m", tagMessage).Run(); err != nil {
		return fmt.Errorf("创建标签失败: %v", err)
	}
	writeSuccess(fmt.Sprintf("标签 %s 已创建", tagName))

	// 推送代码
	writeInfo("推送代码到远程仓库...")
	for i := 0; i < retryConfig.CodePushRetries; i++ {
		if err := exec.Command("git", "push", "origin", "HEAD").Run(); err != nil {
			if i < retryConfig.CodePushRetries-1 {
				writeWarning(fmt.Sprintf("代码推送失败，%d秒后重试... (尝试 %d/%d)", retryConfig.CodePushDelay, i+1, retryConfig.CodePushRetries))
				time.Sleep(time.Duration(retryConfig.CodePushDelay) * time.Second)
				continue
			}
			return fmt.Errorf("代码推送失败: %v", err)
		}
		break
	}
	writeSuccess("代码推送成功")

	// 推送标签
	writeInfo("推送标签到远程仓库...")
	for i := 0; i < retryConfig.TagPushRetries; i++ {
		if err := exec.Command("git", "push", "origin", tagName).Run(); err != nil {
			if i < retryConfig.TagPushRetries-1 {
				writeWarning(fmt.Sprintf("标签推送失败，%d秒后重试... (尝试 %d/%d)", retryConfig.TagPushDelay, i+1, retryConfig.TagPushRetries))
				time.Sleep(time.Duration(retryConfig.TagPushDelay) * time.Second)
				continue
			}
			return fmt.Errorf("标签推送失败: %v", err)
		}
		break
	}
	writeSuccess("标签推送成功")

	return nil
}

// 显示完成信息
func showCompletionMessage(newVersion, owner, repo string, config *Config) {
	writeHeader("🎉 发布完成!")

	fmt.Printf("版本: v%s\n", newVersion)
	fmt.Printf("GitHub仓库: %s/%s\n", owner, repo)
	fmt.Printf("构建状态: https://github.com/%s/%s/actions\n", owner, repo)
	fmt.Printf("发布页面: https://github.com/%s/%s/releases\n", owner, repo)

	if config.Release.AutoOpenBrowser {
		writeInfo("正在打开浏览器...")
		// 这里可以添加打开浏览器的代码
	}

	writeSuccess("🚀 发布流程已完成!")
}

// 主函数
func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run release.go <version> [options]")
		fmt.Println("选项:")
		fmt.Println("  -skip-build    跳过构建过程")
		fmt.Println("  -skip-release  跳过发布过程")
		fmt.Println("  -force         强制执行，跳过确认")
		fmt.Println("")
		fmt.Println("示例:")
		fmt.Println("  go run release.go v1.3.0")
		fmt.Println("  go run release.go 1.3.0 -skip-build")
		fmt.Println("  go run release.go v1.3.0 -force")
		os.Exit(1)
	}

	// 解析命令行参数
	opts := ReleaseOptions{
		Version: os.Args[1],
	}

	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-skip-build":
			opts.SkipBuild = true
		case "-skip-release":
			opts.SkipRelease = true
		case "-force":
			opts.Force = true
		}
	}

	writeHeader("URL Navigator Go Release Tool")

	// 验证版本格式
	newVersion, valid := validateVersionFormat(opts.Version)
	if !valid {
		os.Exit(1)
	}

	writeSuccess(fmt.Sprintf("目标版本: v%s", newVersion))
	writeInfo(fmt.Sprintf("版本号: %s", newVersion))
	writeInfo(fmt.Sprintf("跳过构建: %t", opts.SkipBuild))
	writeInfo(fmt.Sprintf("跳过发布: %t", opts.SkipRelease))

	// 检查前置条件
	if !checkPrerequisites() {
		writeError("前置条件检查失败")
		os.Exit(1)
	}

	// 检查Git仓库状态
	if !checkGitRepository() {
		os.Exit(1)
	}

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		writeError(fmt.Sprintf("加载配置失败: %v", err))
		os.Exit(1)
	}

	// 获取当前版本
	currentVersion, err := getCurrentVersion()
	if err != nil {
		writeError(fmt.Sprintf("获取当前版本失败: %v", err))
		os.Exit(1)
	}

	// 获取GitHub信息
	owner, repo, err := getGitHubInfo(config)
	if err != nil {
		writeError(fmt.Sprintf("获取GitHub信息失败: %v", err))
		os.Exit(1)
	}

	writeInfo(fmt.Sprintf("当前版本: v%s", currentVersion))
	writeInfo(fmt.Sprintf("GitHub仓库: %s/%s", owner, repo))

	// 检查版本是否不同
	if newVersion == currentVersion {
		writeError("新版本不能与当前版本相同")
		os.Exit(1)
	}

	// 确认操作
	if !confirmOperation(currentVersion, newVersion, opts) {
		writeInfo("操作已取消")
		os.Exit(0)
	}

	// 更新版本文件
	if err := updateVersionFiles(newVersion, owner, repo); err != nil {
		writeError(fmt.Sprintf("更新版本文件失败: %v", err))
		os.Exit(1)
	}

	// 构建应用程序（如果未跳过）
	if !opts.SkipBuild {
		if err := buildApplication(newVersion, owner, repo, config); err != nil {
			writeError(fmt.Sprintf("构建失败: %v", err))
			os.Exit(1)
		}
	} else {
		writeInfo("跳过构建步骤")
	}

	// 发布操作（如果未跳过）
	if !opts.SkipRelease {
		if err := performGitOperations(newVersion, config); err != nil {
			writeError(fmt.Sprintf("Git操作失败: %v", err))
			os.Exit(1)
		}
	} else {
		writeInfo("跳过发布步骤")
	}

	// 显示完成信息
	showCompletionMessage(newVersion, owner, repo, config)
}