package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

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

// 获取项目根目录
func getProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 如果当前在tools目录，则返回父目录
	if filepath.Base(currentDir) == "tools" {
		return filepath.Dir(currentDir), nil
	}

	return currentDir, nil
}

// 开发模式
func devMode() error {
	writeHeader("🚀 URL Navigator Development Mode")

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("无法获取项目根目录: %v", err)
	}

	writeInfo("启动开发模式...")
	writeInfo("项目目录: " + projectRoot)

	// 设置工作目录
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("无法切换到项目目录: %v", err)
	}

	// 检查前端依赖
	writeInfo("检查前端依赖...")
	frontendDir := filepath.Join(projectRoot, "frontend")
	if _, err := os.Stat(filepath.Join(frontendDir, "node_modules")); os.IsNotExist(err) {
		writeInfo("安装前端依赖...")
		cmd := exec.Command("yarn", "install")
		cmd.Dir = frontendDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("前端依赖安装失败: %v", err)
		}
		writeSuccess("前端依赖安装完成")
	}

	// 启动wails dev
	writeInfo("启动Wails开发服务器...")
	writeInfo("按 Ctrl+C 停止开发服务器")

	cmd := exec.Command("wails", "dev")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = projectRoot

	// 处理Ctrl+C信号
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		writeInfo("正在停止开发服务器...")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		os.Exit(0)
	}()

	return cmd.Run()
}

// 构建模式
func buildMode() error {
	writeHeader("🔨 URL Navigator Build")

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("无法获取项目根目录: %v", err)
	}

	// 设置工作目录
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("无法切换到项目目录: %v", err)
	}

	// 安装前端依赖
	writeInfo("安装前端依赖...")
	frontendDir := filepath.Join(projectRoot, "frontend")
	cmd := exec.Command("yarn", "install", "--frozen-lockfile")
	cmd.Dir = frontendDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("前端依赖安装失败: %v", err)
	}
	writeSuccess("前端依赖安装完成")

	// 构建前端
	writeInfo("构建前端...")
	cmd = exec.Command("yarn", "build")
	cmd.Dir = frontendDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("前端构建失败: %v", err)
	}
	writeSuccess("前端构建完成")

	// 构建Wails应用
	writeInfo("构建Wails应用...")
	cmd = exec.Command("wails", "build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Wails构建失败: %v", err)
	}

	// 检查构建结果
	exePath := filepath.Join(projectRoot, "build", "bin", "URLNavigator.exe")
	if _, err := os.Stat(exePath); err == nil {
		writeSuccess("应用构建完成: " + exePath)
	} else {
		return fmt.Errorf("构建文件不存在: %s", exePath)
	}

	return nil
}

// 运行应用
func runApp() error {
	writeHeader("▶️  URL Navigator Run")

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("无法获取项目根目录: %v", err)
	}

	exePath := filepath.Join(projectRoot, "build", "bin", "URLNavigator.exe")

	// 检查应用是否存在
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		writeError("应用文件不存在，正在构建...")
		if err := buildMode(); err != nil {
			return fmt.Errorf("构建失败: %v", err)
		}
	}

	writeInfo("启动应用: " + exePath)
	cmd := exec.Command(exePath)
	cmd.Dir = projectRoot

	return cmd.Run()
}

// 简化版发布功能
func releaseMode(version string, skipBuild, skipRelease, force, debug bool) error {
	writeHeader("📦 URL Navigator Release")

	projectRoot, err := getProjectRoot()
	if err != nil {
		return fmt.Errorf("无法获取项目根目录: %v", err)
	}

	// 设置工作目录
	if err := os.Chdir(projectRoot); err != nil {
		return fmt.Errorf("无法切换到项目目录: %v", err)
	}

	// 验证版本格式
	versionRegex := regexp.MustCompile(`^v?(\d+\.\d+\.\d+)$`)
	matches := versionRegex.FindStringSubmatch(version)
	if matches == nil {
		return fmt.Errorf("版本格式无效。请使用 vX.Y.Z 或 X.Y.Z 格式")
	}

	cleanVersion := matches[1]
	versionWithV := "v" + cleanVersion

	writeInfo(fmt.Sprintf("目标版本: %s", versionWithV))
	writeInfo(fmt.Sprintf("跳过构建: %t", skipBuild))
	writeInfo(fmt.Sprintf("跳过发布: %t", skipRelease))

	if debug {
		writeInfo("=== 调试模式已启用 ===")
		// 显示当前Git状态
		if cmd := exec.Command("git", "status", "--porcelain"); true {
			if output, err := cmd.CombinedOutput(); err == nil {
				writeInfo(fmt.Sprintf("Git状态输出:\n%s", string(output)))
			}
		}
	}

	// 检查Git状态
	if !skipRelease {
		writeInfo("检查Git仓库状态...")
		cmd := exec.Command("git", "status", "--porcelain")
		output, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("无法检查Git状态: %v", err)
		}

		if len(strings.TrimSpace(string(output))) > 0 {
			return fmt.Errorf("工作目录有未提交的更改，请先提交")
		}
		writeSuccess("Git仓库状态正常")
	}

	// 确认操作
	if !force {
		fmt.Printf("\n将执行以下操作:\n")
		fmt.Printf("- 更新版本到 %s\n", versionWithV)
		if !skipBuild {
			fmt.Printf("- 构建Windows应用程序\n")
		}
		if !skipRelease {
			fmt.Printf("- 创建Git标签并推送\n")
		}

		fmt.Print("\n确认继续? (y/N): ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		response := strings.ToLower(strings.TrimSpace(scanner.Text()))
		if response != "y" && response != "yes" {
			writeInfo("操作已取消")
			return nil
		}
	}

	// 更新版本文件
	writeInfo("更新版本文件...")
	if err := updateVersionFiles(cleanVersion); err != nil {
		return fmt.Errorf("更新版本文件失败: %v", err)
	}
	writeSuccess("版本文件更新成功")

	// 构建应用
	if !skipBuild {
		if err := buildMode(); err != nil {
			return fmt.Errorf("构建失败: %v", err)
		}
	}

	// 发布操作
	if !skipRelease {
		writeInfo("执行Git操作...")

		// 添加修改的文件
		writeInfo("添加版本文件到Git...")
		cmd := exec.Command("git", "add", "wails.json", "frontend/package.json")
		if output, err := cmd.CombinedOutput(); err != nil {
			writeWarning(fmt.Sprintf("Git add 输出: %s", string(output)))
			return fmt.Errorf("Git add 失败: %v", err)
		}

		// 检查是否有文件需要提交
		cmd = exec.Command("git", "diff", "--cached", "--quiet")
		if err := cmd.Run(); err != nil {
			// 有文件需要提交
			writeInfo("检测到文件变更，执行提交...")

			// 提交更改
			commitMessage := fmt.Sprintf("chore: bump version to %s", versionWithV)
			cmd = exec.Command("git", "commit", "-m", commitMessage)
			if output, err := cmd.CombinedOutput(); err != nil {
				writeWarning(fmt.Sprintf("Git commit 输出: %s", string(output)))
				writeWarning("Git commit 失败，但继续执行后续操作...")
				// 不返回错误，继续执行
			} else {
				writeSuccess("版本更新已提交")
			}
		} else {
			writeInfo("没有文件变更需要提交，跳过commit步骤")
		}

		// 创建标签（即使commit失败也要创建标签）
		writeInfo("创建Git标签...")
		tagMessage := fmt.Sprintf("Release %s", versionWithV)
		cmd = exec.Command("git", "tag", "-a", versionWithV, "-m", tagMessage)
		if output, err := cmd.CombinedOutput(); err != nil {
			// 检查是否是标签已存在的错误
			if strings.Contains(string(output), "already exists") {
				writeWarning(fmt.Sprintf("标签 %s 已存在，删除旧标签重新创建...", versionWithV))
				// 删除旧标签
				exec.Command("git", "tag", "-d", versionWithV).Run()
				exec.Command("git", "push", "origin", ":refs/tags/"+versionWithV).Run()
				// 重新创建标签
				if err := exec.Command("git", "tag", "-a", versionWithV, "-m", tagMessage).Run(); err != nil {
					writeWarning("重新创建标签失败，但继续执行...")
				} else {
					writeSuccess(fmt.Sprintf("标签 %s 已重新创建", versionWithV))
				}
			} else {
				writeWarning(fmt.Sprintf("创建标签输出: %s", string(output)))
				writeWarning("创建标签失败，但继续执行推送操作...")
			}
		} else {
			writeSuccess(fmt.Sprintf("标签 %s 已创建", versionWithV))
		}

		// 推送代码和标签
		writeInfo("推送到远程仓库...")

		// 推送代码
		cmd = exec.Command("git", "push", "origin", "HEAD")
		if output, err := cmd.CombinedOutput(); err != nil {
			writeWarning(fmt.Sprintf("推送代码输出: %s", string(output)))
			writeWarning("推送代码失败，但继续尝试推送标签...")
		} else {
			writeSuccess("代码推送成功")
		}

		// 推送标签
		cmd = exec.Command("git", "push", "origin", versionWithV)
		if output, err := cmd.CombinedOutput(); err != nil {
			writeWarning(fmt.Sprintf("推送标签输出: %s", string(output)))
			writeWarning("推送标签失败，请手动推送标签:")
			writeInfo(fmt.Sprintf("手动命令: git push origin %s", versionWithV))
		} else {
			writeSuccess("标签推送成功")
		}
	}

	writeSuccess("🚀 发布流程完成!")
	return nil
}

// 更新版本文件
func updateVersionFiles(version string) error {
	// 更新wails.json
	if data, err := os.ReadFile("wails.json"); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			if config["info"] == nil {
				config["info"] = make(map[string]interface{})
			}
			info := config["info"].(map[string]interface{})
			info["version"] = version

			if updatedData, err := json.MarshalIndent(config, "", "  "); err == nil {
				os.WriteFile("wails.json", updatedData, 0644)
				writeSuccess("更新wails.json成功")
			}
		}
	}

	// 更新frontend/package.json
	packagePath := "frontend/package.json"
	if data, err := os.ReadFile(packagePath); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(data, &config); err == nil {
			config["version"] = version

			if updatedData, err := json.MarshalIndent(config, "", "  "); err == nil {
				os.WriteFile(packagePath, updatedData, 0644)
				writeSuccess("更新frontend/package.json成功")
			}
		}
	}

	return nil
}

// 显示帮助信息
func showHelp() {
	writeHeader("URL Navigator Development & Release Tool")

	fmt.Println("用法: go run tools/urlnav.go <command> [options]")
	fmt.Println()
	fmt.Println("开发命令:")
	fmt.Println("  dev      启动开发模式 (wails dev)")
	fmt.Println("  build    构建应用程序")
	fmt.Println("  run      运行构建的应用程序")
	fmt.Println()
	fmt.Println("发布命令:")
	fmt.Println("  release  发布新版本")
	fmt.Println()
	fmt.Println("其他命令:")
	fmt.Println("  help     显示此帮助信息")
	fmt.Println()
	fmt.Println("开发示例:")
	fmt.Println("  go run tools/urlnav.go dev")
	fmt.Println("  go run tools/urlnav.go build")
	fmt.Println("  go run tools/urlnav.go run")
	fmt.Println()
	fmt.Println("发布示例:")
	fmt.Println("  go run tools/urlnav.go release v1.4.0")
	fmt.Println("  go run tools/urlnav.go release v1.4.0 -skip-build")
	fmt.Println()
	fmt.Println("发布选项:")
	fmt.Println("  -skip-build    跳过构建过程")
	fmt.Println("  -skip-release  跳过发布过程")
	fmt.Println("  -force         强制执行，跳过确认")
	fmt.Println("  -debug         启用调试模式，显示详细信息")
}

func main() {
	if len(os.Args) < 2 {
		showHelp()
		os.Exit(1)
	}

	command := os.Args[1]

	var err error
	switch command {
	case "dev":
		err = devMode()
	case "build":
		err = buildMode()
	case "run":
		err = runApp()
	case "release":
		if len(os.Args) < 3 {
			writeError("发布命令需要版本号参数")
			showHelp()
			os.Exit(1)
		}

		version := os.Args[2]
		skipBuild := false
		skipRelease := false
		force := false
		debug := false

		// 解析选项
		for i := 3; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "-skip-build":
				skipBuild = true
			case "-skip-release":
				skipRelease = true
			case "-force":
				force = true
			case "-debug":
				debug = true
			}
		}

		err = releaseMode(version, skipBuild, skipRelease, force, debug)
	case "help", "-h", "--help":
		showHelp()
		return
	default:
		writeError("未知命令: " + command)
		showHelp()
		os.Exit(1)
	}

	if err != nil {
		writeError(fmt.Sprintf("命令执行失败: %v", err))
		os.Exit(1)
	}
}