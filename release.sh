#!/bin/bash

# URL Navigator Windows版本发布脚本
# 用于创建新版本的标签并触发GitHub Actions构建

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印彩色消息
print_message() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示欢迎信息
echo "=================================="
echo "URL Navigator Windows 版本发布工具"
echo "=================================="
echo

# 检查是否在git仓库中
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "当前目录不是git仓库"
    exit 1
fi

# 检查工作区是否干净
if [ -n "$(git status --porcelain)" ]; then
    print_error "工作区有未提交的改动，请先提交或储藏改动"
    echo
    git status --short
    exit 1
fi

# 获取当前版本
CURRENT_VERSION=$(grep '"version":' wails.json | sed 's/.*"version": *"\([^"]*\)".*/\1/')
if [ -z "$CURRENT_VERSION" ]; then
    print_error "无法从wails.json中读取版本号"
    exit 1
fi

print_message "当前版本: $CURRENT_VERSION"

# 获取新版本号
if [ -n "$1" ]; then
    NEW_VERSION="$1"
else
    echo -n "请输入新版本号 (当前: $CURRENT_VERSION): "
    read NEW_VERSION
fi

# 验证版本号格式
if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
    print_error "版本号格式错误，应为 x.y.z 格式 (例如: 1.2.0)"
    exit 1
fi

# 检查版本号是否比当前版本新
if [ "$NEW_VERSION" = "$CURRENT_VERSION" ]; then
    print_error "新版本号不能与当前版本号相同"
    exit 1
fi

print_message "新版本: $NEW_VERSION"
print_message "目标平台: Windows x64"

# 确认发布
echo
echo -n "确认发布 Windows 版本 $NEW_VERSION? (y/N): "
read CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    print_warning "发布已取消"
    exit 0
fi

echo
print_message "开始发布流程..."

# 更新版本号
print_message "更新版本号..."

# 更新 wails.json
if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    sed -i '' "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" wails.json
else
    # Linux/Windows (Git Bash)
    sed -i "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" wails.json
fi

# 更新 updater.go 中的 CurrentVersion
if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "s/CurrentVersion = \"[^\"]*\"/CurrentVersion = \"$NEW_VERSION\"/" updater.go
else
    sed -i "s/CurrentVersion = \"[^\"]*\"/CurrentVersion = \"$NEW_VERSION\"/" updater.go
fi

# 更新 frontend/package.json
if [ -f "frontend/package.json" ]; then
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
    else
        sed -i "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
    fi
fi

print_success "版本号已更新"

# 显示变更
echo
print_message "版本更新摘要:"
echo "  - wails.json: $CURRENT_VERSION → $NEW_VERSION"
echo "  - updater.go: CurrentVersion = \"$NEW_VERSION\""
if [ -f "frontend/package.json" ]; then
    echo "  - frontend/package.json: $NEW_VERSION"
fi

# 提交更改
echo
print_message "提交版本更新..."
git add wails.json updater.go 2>/dev/null || true
if [ -f "frontend/package.json" ]; then
    git add frontend/package.json 2>/dev/null || true
fi

git commit -m "chore: bump version to $NEW_VERSION

- Update version in wails.json
- Update CurrentVersion in updater.go
- Prepare for Windows release"

print_success "版本更新已提交"

# 创建标签
echo
print_message "创建标签 v$NEW_VERSION..."
git tag -a "v$NEW_VERSION" -m "Release Windows version $NEW_VERSION

Features:
- Bookmark management
- Category system
- Auto-update functionality
- Modern UI with shadcn/ui

Platform: Windows x64"

print_success "标签已创建"

# 推送到远程仓库
echo
print_message "推送到远程仓库..."
# 尝试推送到main分支，如果失败则尝试master分支
if git push origin main 2>/dev/null; then
    print_success "代码已推送到 main 分支"
elif git push origin master 2>/dev/null; then
    print_success "代码已推送到 master 分支"
else
    print_error "推送失败，请检查远程仓库配置"
    exit 1
fi

print_message "推送标签..."
git push origin "v$NEW_VERSION"
print_success "标签已推送"

# 完成提示
echo
echo "🎉 =================================="
print_success "Windows 版本 $NEW_VERSION 发布完成!"
echo "===================================="
echo

print_message "接下来将发生什么:"
echo "  1. GitHub Actions 将自动开始构建 Windows 版本"
echo "  2. 构建完成后将自动创建 GitHub Release"
echo "  3. Release 中将包含 URLNavigator.exe 文件"
echo "  4. 应用的自动更新功能将检测到新版本"

echo
print_message "监控链接:"
echo "  - 构建状态: https://github.com/wangyaxings/url-navigator/actions"
echo "  - 发布页面: https://github.com/wangyaxings/url-navigator/releases"

# 等待几秒后尝试打开浏览器
if command -v start >/dev/null 2>&1; then
    # Windows (Git Bash)
    print_message "3秒后将打开GitHub Actions页面..."
    sleep 3
    start "https://github.com/wangyaxings/url-navigator/actions"
elif command -v open >/dev/null 2>&1; then
    # macOS
    print_message "3秒后将打开GitHub Actions页面..."
    sleep 3
    open "https://github.com/wangyaxings/url-navigator/actions"
elif command -v xdg-open >/dev/null 2>&1; then
    # Linux
    print_message "3秒后将打开GitHub Actions页面..."
    sleep 3
    xdg-open "https://github.com/wangyaxings/url-navigator/actions"
fi

echo
print_message "发布流程完成! 🚀"