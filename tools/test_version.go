package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// 导入父目录的版本管理代码
// 需要复制相关结构和函数到这里进行测试

func main() {
	fmt.Println("=== 版本信息测试 ===")

		// 获取当前工作目录
	currentDir, _ := os.Getwd()
	fmt.Printf("当前目录: %s\n", currentDir)

	// 项目根目录应该是当前目录的父目录
	projectRoot := filepath.Dir(currentDir)
	dataDir := filepath.Join(os.Getenv("USERPROFILE"), ".urlnavigator")

	fmt.Printf("项目根目录: %s\n", projectRoot)
	fmt.Printf("数据目录: %s\n", dataDir)

	// 测试读取wails.json版本
	fmt.Println("\n1. 读取wails.json版本:")
	wailsPath := filepath.Join(projectRoot, "wails.json")
	if content, err := os.ReadFile(wailsPath); err == nil {
		fmt.Printf("wails.json 内容前100字符: %s...\n", string(content)[:100])
	} else {
		fmt.Printf("读取wails.json失败: %v\n", err)
	}

	// 测试读取version.json版本
	fmt.Println("\n2. 读取version.json版本:")
	versionPath := filepath.Join(projectRoot, "version.json")
	if content, err := os.ReadFile(versionPath); err == nil {
		fmt.Printf("version.json 内容前100字符: %s...\n", string(content)[:100])
	} else {
		fmt.Printf("读取version.json失败: %v\n", err)
	}

	fmt.Println("\n测试完成")
}