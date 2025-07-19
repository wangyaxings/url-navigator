#!/bin/bash

# URL Navigator 发布脚本
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

# 检查是否在git仓库中
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    print_error "当前目录不是git仓库"
    exit 1
fi

# 检查工作区是否干净
if [ -n "$(git status --porcelain)" ]; then
    print_error "工作区有未提交的改动，请先提交或储藏改动"
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
    print_error "版本号格式错误，应为 x.y.z 格式"
    exit 1
fi

# 检查版本号是否比当前版本新
if [ "$NEW_VERSION" = "$CURRENT_VERSION" ]; then
    print_error "新版本号不能与当前版本号相同"
    exit 1
fi

print_message "新版本: $NEW_VERSION"

# 确认发布
echo -n "确认发布版本 $NEW_VERSION? (y/N): "
read CONFIRM
if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
    print_warning "发布已取消"
    exit 0
fi

# 更新版本号
print_message "更新版本号..."

# 更新 wails.json
sed -i.bak "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" wails.json
rm -f wails.json.bak

# 更新 updater.go 中的 CurrentVersion
sed -i.bak "s/CurrentVersion = \"[^\"]*\"/CurrentVersion = \"$NEW_VERSION\"/" updater.go
rm -f updater.go.bak

# 更新 frontend/package.json
if [ -f "frontend/package.json" ]; then
    sed -i.bak "s/\"version\": *\"[^\"]*\"/\"version\": \"$NEW_VERSION\"/" frontend/package.json
    rm -f frontend/package.json.bak
fi

print_success "版本号已更新"

# 提交更改
print_message "提交版本更新..."
git add wails.json updater.go frontend/package.json 2>/dev/null || true
git commit -m "chore: bump version to $NEW_VERSION"

# 创建标签
print_message "创建标签 v$NEW_VERSION..."
git tag -a "v$NEW_VERSION" -m "Release version $NEW_VERSION"

# 推送到远程仓库
print_message "推送到远程仓库..."
git push origin main || git push origin master
git push origin "v$NEW_VERSION"

print_success "版本 $NEW_VERSION 发布完成!"
print_message "GitHub Actions 将自动构建并创建 Release"
print_message "查看构建状态: https://github.com/wangyaxings/url-navigator/actions"

# 等待几秒后打开浏览器
if command -v open >/dev/null 2>&1; then
    print_message "3秒后将打开GitHub Actions页面..."
    sleep 3
    open "https://github.com/wangyaxings/url-navigator/actions"
elif command -v xdg-open >/dev/null 2>&1; then
    print_message "3秒后将打开GitHub Actions页面..."
    sleep 3
    xdg-open "https://github.com/wangyaxings/url-navigator/actions"
fi