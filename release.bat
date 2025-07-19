@echo off
setlocal enabledelayedexpansion

:: URL Navigator Windows版本发布脚本
:: 用于自动构建、版本管理和发布

title URL Navigator Release Builder

:: 颜色设置
:: 由于批处理的颜色较复杂，使用echo输出清晰信息

echo ======================================
echo URL Navigator Windows 版本发布工具
echo ======================================
echo.

:: 检查参数
if "%~1"=="" (
    echo 用法: %0 ^<新版本号^> [--skip-build] [--skip-release]
    echo 示例: %0 1.3.0
    echo       %0 1.3.0 --skip-build    ^# 跳过构建，仅创建 release
    echo       %0 1.3.0 --skip-release  ^# 仅构建，不创建 release
    echo.
    pause
    exit /b 1
)

set NEW_VERSION=%1
set SKIP_BUILD=false
set SKIP_RELEASE=false

:: 解析参数
:parse_args
if "%~2"=="--skip-build" set SKIP_BUILD=true
if "%~2"=="--skip-release" set SKIP_RELEASE=true
if "%~3"=="--skip-build" set SKIP_BUILD=true
if "%~3"=="--skip-release" set SKIP_RELEASE=true

:: 验证版本号格式
echo %NEW_VERSION% | findstr /r "^[0-9]*\.[0-9]*\.[0-9]*$" >nul
if errorlevel 1 (
    echo [ERROR] 版本号格式错误，应为 x.y.z 格式 ^(例如: 1.3.0^)
    pause
    exit /b 1
)

echo [INFO] 目标版本: %NEW_VERSION%
echo [INFO] 跳过构建: %SKIP_BUILD%
echo [INFO] 跳过发布: %SKIP_RELEASE%
echo.

:: 检查必要工具
echo [INFO] 检查必要工具...

:: 检查Git
git --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Git 未安装或未在PATH中
    pause
    exit /b 1
)

:: 检查Go
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go 未安装或未在PATH中
    pause
    exit /b 1
)

:: 检查Wails
wails version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Wails CLI 未安装，请运行: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    pause
    exit /b 1
)

:: 检查Yarn
yarn --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Yarn 未安装或未在PATH中
    pause
    exit /b 1
)

echo [SUCCESS] 所有必要工具已安装
echo.

:: 检查Git仓库状态
echo [INFO] 检查Git仓库状态...

git rev-parse --git-dir >nul 2>&1
if errorlevel 1 (
    echo [ERROR] 当前目录不是Git仓库
    pause
    exit /b 1
)

:: 检查工作区是否干净
git status --porcelain 2>nul | findstr /r /v "^$" >nul
if not errorlevel 1 (
    echo [ERROR] 工作区有未提交的改动，请先提交或储藏改动
    git status --short
    pause
    exit /b 1
)

echo [SUCCESS] Git仓库状态正常
echo.

:: 获取当前版本
echo [INFO] 获取当前版本信息...

if not exist "wails.json" (
    echo [ERROR] wails.json 文件不存在
    pause
    exit /b 1
)

:: 使用PowerShell解析JSON（更可靠）
for /f "usebackq delims=" %%i in (`powershell -command "(Get-Content 'wails.json' | ConvertFrom-Json).info.version"`) do set CURRENT_VERSION=%%i

if "%CURRENT_VERSION%"=="" (
    echo [ERROR] 无法从wails.json中读取版本号
    pause
    exit /b 1
)

echo [INFO] 当前版本: %CURRENT_VERSION%
echo [INFO] 新版本: %NEW_VERSION%

:: 检查版本号
if "%NEW_VERSION%"=="%CURRENT_VERSION%" (
    echo [ERROR] 新版本号不能与当前版本号相同
    pause
    exit /b 1
)

:: 获取GitHub仓库信息
echo [INFO] 获取GitHub仓库信息...

for /f "usebackq delims=" %%i in (`git config --get remote.origin.url`) do set REMOTE_URL=%%i

:: 解析GitHub仓库信息（简化版本）
echo %REMOTE_URL% | findstr "github.com" >nul
if errorlevel 1 (
    echo [WARNING] 无法识别GitHub仓库，请手动输入
    set /p GITHUB_OWNER="请输入GitHub用户名: "
    set /p GITHUB_REPO="请输入仓库名: "
) else (
    :: 从URL中提取用户名和仓库名（需要进一步解析）
    echo [INFO] 检测到GitHub仓库: %REMOTE_URL%
    :: 这里可以添加更复杂的URL解析逻辑
    set GITHUB_OWNER=wangyaxings
    set GITHUB_REPO=url-navigator
)

echo [INFO] GitHub仓库: %GITHUB_OWNER%/%GITHUB_REPO%
echo.

:: 确认操作
echo 即将执行以下操作：
echo - 更新版本号从 %CURRENT_VERSION% 到 %NEW_VERSION%
echo - 构建Windows应用程序
echo - 创建Git标签并推送
echo - 创建GitHub Release
echo.
set /p CONFIRM="确认继续? (y/N): "
if /i not "%CONFIRM%"=="y" (
    echo [INFO] 操作已取消
    pause
    exit /b 0
)

echo.
echo [INFO] 开始发布流程...
echo.

:: 更新版本号
echo [INFO] 更新版本号...

:: 备份原文件
copy wails.json wails.json.backup >nul
if exist "frontend\package.json" copy frontend\package.json frontend\package.json.backup >nul

:: 使用PowerShell更新wails.json
powershell -command "$json = Get-Content 'wails.json' | ConvertFrom-Json; $json.info.version = '%NEW_VERSION%'; $json.github.owner = '%GITHUB_OWNER%'; $json.github.repo = '%GITHUB_REPO%'; $json | ConvertTo-Json -Depth 10 | Set-Content 'wails.json'"

:: 更新frontend/package.json（如果存在）
if exist "frontend\package.json" (
    powershell -command "$json = Get-Content 'frontend\package.json' | ConvertFrom-Json; $json.version = '%NEW_VERSION%'; $json | ConvertTo-Json -Depth 10 | Set-Content 'frontend\package.json'"
)

echo [SUCCESS] 版本号已更新
echo.

:: 跳过构建检查
if "%SKIP_BUILD%"=="true" (
    echo [INFO] 跳过构建步骤
    goto commit_changes
)

:: 构建应用
echo [INFO] 开始构建应用...

:: 安装前端依赖
echo [INFO] 安装前端依赖...
cd frontend
yarn install
if errorlevel 1 (
    echo [ERROR] 前端依赖安装失败
    cd ..
    pause
    exit /b 1
)
cd ..

:: 构建前端
echo [INFO] 构建前端...
cd frontend
yarn build
if errorlevel 1 (
    echo [ERROR] 前端构建失败
    cd ..
    pause
    exit /b 1
)
cd ..

:: 构建Wails应用
echo [INFO] 构建Wails应用...
wails build -platform windows/amd64 -ldflags "-H=windowsgui -s -w -X main.Version=%NEW_VERSION% -X main.GitHubOwner=%GITHUB_OWNER% -X main.GitHubRepo=%GITHUB_REPO%" -tags production -trimpath
if errorlevel 1 (
    echo [ERROR] Wails应用构建失败
    pause
    exit /b 1
)

:: 检查构建结果
if not exist "build\bin\URLNavigator.exe" (
    echo [ERROR] 构建文件不存在: build\bin\URLNavigator.exe
    pause
    exit /b 1
)

echo [SUCCESS] 应用构建成功
dir "build\bin\URLNavigator.exe"
echo.

:commit_changes
:: 提交版本更新
echo [INFO] 提交版本更新...
git add wails.json
if exist "frontend\package.json" git add frontend\package.json
git commit -m "chore: bump version to %NEW_VERSION%

- Update version in wails.json
- Update frontend package.json
- Prepare for Windows release"

if errorlevel 1 (
    echo [ERROR] 提交失败
    pause
    exit /b 1
)

echo [SUCCESS] 版本更新已提交
echo.

:: 跳过发布检查
if "%SKIP_RELEASE%"=="true" (
    echo [INFO] 跳过发布步骤
    goto finish
)

:: 创建标签
echo [INFO] 创建标签 v%NEW_VERSION%...
git tag -a "v%NEW_VERSION%" -m "Release %NEW_VERSION%

Features:
- Bookmark management with categories
- Auto-update functionality
- Modern UI with shadcn/ui
- Local data storage

Platform: Windows x64
Build: Automated release with version injection"

if errorlevel 1 (
    echo [ERROR] 创建标签失败
    pause
    exit /b 1
)

echo [SUCCESS] 标签已创建
echo.

:: 推送到远程仓库
echo [INFO] 推送到远程仓库...

:: 尝试推送到main分支
git push origin main >nul 2>&1
if not errorlevel 1 (
    echo [SUCCESS] 代码已推送到 main 分支
) else (
    :: 尝试推送到master分支
    git push origin master >nul 2>&1
    if not errorlevel 1 (
        echo [SUCCESS] 代码已推送到 master 分支
    ) else (
        echo [ERROR] 推送失败，请检查远程仓库配置
        pause
        exit /b 1
    )
)

echo [INFO] 推送标签...
git push origin "v%NEW_VERSION%"
if errorlevel 1 (
    echo [ERROR] 标签推送失败
    pause
    exit /b 1
)

echo [SUCCESS] 标签已推送
echo.

:finish
:: 完成提示
echo.
echo ======================================
echo [SUCCESS] Windows 版本 %NEW_VERSION% 发布完成!
echo ======================================
echo.

echo 接下来将发生什么:
echo   1. GitHub Actions 将自动开始构建 Windows 版本
echo   2. 构建完成后将自动创建 GitHub Release
echo   3. Release 中将包含 URLNavigator.exe 文件
echo   4. 应用的自动更新功能将检测到新版本
echo.

echo 监控链接:
echo   - 构建状态: https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/actions
echo   - 发布页面: https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/releases
echo.

:: 等待并打开浏览器
echo [INFO] 3秒后将打开GitHub Actions页面...
timeout /t 3 /nobreak >nul
start "" "https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/actions"

echo.
echo [INFO] 发布流程完成! 🚀
echo.
pause