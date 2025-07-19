#!/bin/bash

# URL Navigator 构建和发布脚本
# 使用统一的版本管理和自动化构建流程

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 输出函数
print_error() {
    echo -e "${RED}❌ $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_message() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_header() {
    echo
    echo "🚀 ========================================"
    echo -e "${BLUE}   $1${NC}"
    echo "========================================"
    echo
}

# 检查必要的工具
check_prerequisites() {
    print_header "检查构建环境"
    
    # 检查 git
    if ! command -v git &> /dev/null; then
        print_error "Git 未安装"
        exit 1
    fi
    
    # 检查 go
    if ! command -v go &> /dev/null; then
        print_error "Go 未安装"
        exit 1
    fi
    
    # 检查 wails
    if ! command -v wails &> /dev/null; then
        print_error "Wails 未安装，请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    fi
    
    # 检查 yarn (前端依赖)
    if ! command -v yarn &> /dev/null; then
        print_error "Yarn 未安装"
        exit 1
    fi
    
    print_success "所有必要工具已安装"
}

# 获取当前版本
get_current_version() {
    # 从 wails.json 获取当前版本
    if [ -f "wails.json" ]; then
        CURRENT_VERSION=$(grep -o '"version": *"[^"]*"' wails.json | sed 's/.*: *"\([^"]*\)".*/\1/')
    else
        print_error "未找到 wails.json 文件"
        exit 1
    fi
    
    if [ -z "$CURRENT_VERSION" ]; then
        print_error "无法从 wails.json 获取版本信息"
        exit 1
    fi
}

# 获取Git仓库信息
get_git_info() {
    # 尝试从 git remote 获取仓库信息
    if git remote -v &> /dev/null; then
        REMOTE_URL=$(git remote get-url origin 2>/dev/null || echo "")
        if [[ "$REMOTE_URL" =~ github\.com[:/]([^/]+)/([^/.]+) ]]; then
            GITHUB_OWNER="${BASH_REMATCH[1]}"
            GITHUB_REPO="${BASH_REMATCH[2]}"
            GITHUB_REPO="${GITHUB_REPO%.git}" # 移除 .git 后缀
        else
            print_warning "无法从 Git remote 解析 GitHub 仓库信息"
            # 提示用户手动输入
            read -p "请输入 GitHub 用户名: " GITHUB_OWNER
            read -p "请输入仓库名: " GITHUB_REPO
        fi
    else
        print_warning "不是 Git 仓库或没有配置 remote"
        read -p "请输入 GitHub 用户名: " GITHUB_OWNER
        read -p "请输入仓库名: " GITHUB_REPO
    fi
}

# 更新版本号
update_version() {
    local new_version="$1"
    
    print_message "更新版本号到 $new_version..."
    
    # 更新 wails.json
    if [[ "$OSTYPE" == "darwin"* ]]; then
        sed -i '' "s/\"version\": *\"[^\"]*\"/\"version\": \"$new_version\"/" wails.json
    else
        sed -i "s/\"version\": *\"[^\"]*\"/\"version\": \"$new_version\"/" wails.json
    fi
    
    # 更新 frontend/package.json（如果存在）
    if [ -f "frontend/package.json" ]; then
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s/\"version\": *\"[^\"]*\"/\"version\": \"$new_version\"/" frontend/package.json
        else
            sed -i "s/\"version\": *\"[^\"]*\"/\"version\": \"$new_version\"/" frontend/package.json
        fi
    fi
    
    print_success "版本号已更新"
}

# 构建应用
build_app() {
    print_header "构建应用"
    
    # 安装前端依赖
    print_message "安装前端依赖..."
    cd frontend
    yarn install
    cd ..
    
    # 构建 Windows 版本（主要目标平台）
    print_message "构建 Windows 版本..."
    wails build -platform windows/amd64 \
        -ldflags "-X main.Version=$CURRENT_VERSION -X main.GitHubOwner=$GITHUB_OWNER -X main.GitHubRepo=$GITHUB_REPO" \
        -compress
    
    # 检查构建结果
    if [ -f "build/bin/URLNavigator.exe" ]; then
        print_success "Windows 版本构建成功"
        ls -la build/bin/URLNavigator.exe
    else
        print_error "Windows 版本构建失败"
        exit 1
    fi
}

# 创建 GitHub Release
create_github_release() {
    local version="$1"
    local tag_name="v$version"
    
    print_header "创建 GitHub Release"
    
    # 检查是否安装了 gh CLI
    if ! command -v gh &> /dev/null; then
        print_warning "GitHub CLI (gh) 未安装，将手动创建 release"
        manual_release_instructions "$version" "$tag_name"
        return
    fi
    
    # 检查是否已登录 GitHub
    if ! gh auth status &> /dev/null; then
        print_warning "未登录 GitHub CLI，将手动创建 release"
        manual_release_instructions "$version" "$tag_name"
        return
    fi
    
    # 创建标签
    print_message "创建标签 $tag_name..."
    git tag -a "$tag_name" -m "Release $version

Features:
- Bookmark management with categories
- Auto-update functionality
- Modern UI with shadcn/ui
- Local data storage

Platform: Windows x64
Build: Automated release with version injection"
    
    # 推送标签
    print_message "推送标签到远程仓库..."
    git push origin "$tag_name"
    
    # 创建 GitHub Release
    print_message "创建 GitHub Release..."
    gh release create "$tag_name" \
        --title "URL Navigator $version" \
        --notes "Release $version

## 新功能
- 书签管理和分类系统
- 自动更新功能
- 现代化 UI 界面
- 本地数据存储

## 安装说明
1. 下载 URLNavigator.exe
2. 双击运行即可使用
3. 应用会自动检查更新

## 系统要求
- Windows 10/11 (x64)
- 无需额外依赖" \
        "build/bin/URLNavigator.exe"
    
    print_success "GitHub Release 创建成功!"
    print_message "Release URL: https://github.com/$GITHUB_OWNER/$GITHUB_REPO/releases/tag/$tag_name"
}

# 手动创建 release 的说明
manual_release_instructions() {
    local version="$1"
    local tag_name="$2"
    
    echo
    print_warning "请手动完成以下步骤:"
    echo
    echo "1. 创建并推送标签:"
    echo "   git tag -a '$tag_name' -m 'Release $version'"
    echo "   git push origin '$tag_name'"
    echo
    echo "2. 在 GitHub 上创建 Release:"
    echo "   - 访问: https://github.com/$GITHUB_OWNER/$GITHUB_REPO/releases/new"
    echo "   - Tag: $tag_name"
    echo "   - Title: URL Navigator $version"
    echo "   - 上传文件: build/bin/URLNavigator.exe"
    echo
}

# 主函数
main() {
    print_header "URL Navigator 自动构建和发布"
    
    # 检查参数
    if [ $# -eq 0 ]; then
        echo "用法: $0 <新版本号> [--skip-build] [--skip-release]"
        echo "示例: $0 1.3.0"
        echo "      $0 1.3.0 --skip-build    # 跳过构建，仅创建 release"
        echo "      $0 1.3.0 --skip-release  # 仅构建，不创建 release"
        exit 1
    fi
    
    NEW_VERSION="$1"
    SKIP_BUILD=false
    SKIP_RELEASE=false
    
    # 解析参数
    for arg in "${@:2}"; do
        case $arg in
            --skip-build)
                SKIP_BUILD=true
                ;;
            --skip-release)
                SKIP_RELEASE=true
                ;;
        esac
    done
    
    # 验证版本号格式
    if ! echo "$NEW_VERSION" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$'; then
        print_error "版本号格式错误，应为 x.y.z 格式 (例如: 1.3.0)"
        exit 1
    fi
    
    # 检查环境
    check_prerequisites
    
    # 获取当前版本和仓库信息
    get_current_version
    get_git_info
    
    print_message "当前版本: $CURRENT_VERSION"
    print_message "新版本: $NEW_VERSION"
    print_message "GitHub仓库: $GITHUB_OWNER/$GITHUB_REPO"
    
    # 检查版本号
    if [ "$NEW_VERSION" = "$CURRENT_VERSION" ]; then
        print_error "新版本号不能与当前版本号相同"
        exit 1
    fi
    
    # 确认操作
    echo
    echo -n "确认继续? (y/N): "
    read CONFIRM
    if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
        print_warning "操作已取消"
        exit 0
    fi
    
    # 更新版本号
    update_version "$NEW_VERSION"
    CURRENT_VERSION="$NEW_VERSION"  # 更新当前版本变量
    
    # 提交版本更新
    print_message "提交版本更新..."
    git add wails.json
    [ -f "frontend/package.json" ] && git add frontend/package.json
    git commit -m "chore: bump version to $NEW_VERSION

- Update version in wails.json
- Prepare for release $NEW_VERSION
- Auto-update functionality ready"
    
    # 构建应用
    if [ "$SKIP_BUILD" = false ]; then
        build_app
    else
        print_warning "跳过构建步骤"
    fi
    
    # 创建 GitHub Release
    if [ "$SKIP_RELEASE" = false ]; then
        create_github_release "$NEW_VERSION"
    else
        print_warning "跳过 Release 创建"
    fi
    
    print_header "构建完成!"
    print_success "版本 $NEW_VERSION 构建和发布流程完成"
    
    if [ "$SKIP_BUILD" = false ]; then
        echo "📦 构建文件: build/bin/URLNavigator.exe"
    fi
    
    if [ "$SKIP_RELEASE" = false ]; then
        echo "🌐 GitHub Release: https://github.com/$GITHUB_OWNER/$GITHUB_REPO/releases"
    fi
    
    echo
    print_message "接下来的步骤:"
    echo "  1. 验证应用程序正常启动和运行"
    echo "  2. 测试自动更新功能"
    echo "  3. 在不同 Windows 系统上测试"
    echo
}

# 运行主函数
main "$@"