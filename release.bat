@echo off
setlocal enabledelayedexpansion

:: URL Navigator Windows Release Script
:: Automated build, version management and release

title URL Navigator Release Builder

echo ======================================
echo URL Navigator Windows Release Tool
echo ======================================
echo.

:: Check parameters
if "%~1"=="" (
    echo Usage: %0 ^<new-version^> [--skip-build] [--skip-release]
    echo Examples: %0 v1.3.0
    echo           %0 v1.3.0 --skip-build    ^# Skip build, only create release
    echo           %0 v1.3.0 --skip-release  ^# Only build, don't create release
    echo.
    echo Note: Version format should be vX.Y.Z ^(e.g., v1.3.0^)
    echo.
    pause
    exit /b 1
)

set NEW_VERSION=%1
set SKIP_BUILD=false
set SKIP_RELEASE=false

:: Parse arguments
:parse_args
if "%~2"=="--skip-build" set SKIP_BUILD=true
if "%~2"=="--skip-release" set SKIP_RELEASE=true
if "%~3"=="--skip-build" set SKIP_BUILD=true
if "%~3"=="--skip-release" set SKIP_RELEASE=true

:: Validate version format (support both vX.Y.Z and X.Y.Z)
echo %NEW_VERSION% | findstr /r "^v[0-9]*\.[0-9]*\.[0-9]*$" >nul
if not errorlevel 1 (
    :: Version with v prefix - remove v for internal processing
    set VERSION_NUMBER=%NEW_VERSION:~1%
) else (
    :: Check if it's X.Y.Z format
    echo %NEW_VERSION% | findstr /r "^[0-9]*\.[0-9]*\.[0-9]*$" >nul
    if not errorlevel 1 (
        :: Version without v prefix - add v for consistency
        set NEW_VERSION=v%NEW_VERSION%
        set VERSION_NUMBER=%NEW_VERSION:~1%
    ) else (
        echo [ERROR] Invalid version format. Use vX.Y.Z or X.Y.Z format ^(e.g., v1.3.0^)
        pause
        exit /b 1
    )
)

echo [INFO] Target version: %NEW_VERSION%
echo [INFO] Version number: %VERSION_NUMBER%
echo [INFO] Skip build: %SKIP_BUILD%
echo [INFO] Skip release: %SKIP_RELEASE%
echo.

:: Check required tools
echo [INFO] Checking required tools...

:: Check Git
git --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Git is not installed or not in PATH
    pause
    exit /b 1
)

:: Check Go
go version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Go is not installed or not in PATH
    pause
    exit /b 1
)

:: Check Wails
wails version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Wails CLI is not installed. Run: go install github.com/wailsapp/wails/v2/cmd/wails@latest
    pause
    exit /b 1
)

:: Check Yarn
yarn --version >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Yarn is not installed or not in PATH
    pause
    exit /b 1
)

echo [SUCCESS] All required tools are installed
echo.

:: Check Git repository status
echo [INFO] Checking Git repository status...

git rev-parse --git-dir >nul 2>&1
if errorlevel 1 (
    echo [ERROR] Current directory is not a Git repository
    pause
    exit /b 1
)

:: Check if working directory is clean
git status --porcelain 2>nul | findstr /r /v "^$" >nul
if not errorlevel 1 (
    echo [ERROR] Working directory has uncommitted changes. Please commit or stash changes first.
    git status --short
    pause
    exit /b 1
)

echo [SUCCESS] Git repository status is clean
echo.

:: Get current version
echo [INFO] Getting current version information...

if not exist "wails.json" (
    echo [ERROR] wails.json file does not exist
    pause
    exit /b 1
)

:: Use PowerShell to parse JSON (more reliable)
for /f "usebackq delims=" %%i in (`powershell -command "(Get-Content 'wails.json' | ConvertFrom-Json).info.version"`) do set CURRENT_VERSION_NUMBER=%%i

if "%CURRENT_VERSION_NUMBER%"=="" (
    echo [ERROR] Cannot read version from wails.json
    pause
    exit /b 1
)

:: Ensure current version has v prefix for display
set CURRENT_VERSION=v%CURRENT_VERSION_NUMBER%

echo [INFO] Current version: %CURRENT_VERSION%
echo [INFO] New version: %NEW_VERSION%

:: Check version numbers
if "%VERSION_NUMBER%"=="%CURRENT_VERSION_NUMBER%" (
    echo [ERROR] New version cannot be the same as current version
    pause
    exit /b 1
)

:: Get GitHub repository information
echo [INFO] Getting GitHub repository information...

for /f "usebackq delims=" %%i in (`git config --get remote.origin.url`) do set REMOTE_URL=%%i

:: Parse GitHub repository information
echo %REMOTE_URL% | findstr "github.com" >nul
if errorlevel 1 (
    echo [WARNING] Cannot identify GitHub repository. Please enter manually.
    set /p GITHUB_OWNER="Enter GitHub username: "
    set /p GITHUB_REPO="Enter repository name: "
) else (
    :: Extract username and repository name from URL
    echo [INFO] Detected GitHub repository: %REMOTE_URL%
    :: For now, use known values. Can be enhanced with URL parsing logic
    set GITHUB_OWNER=wangyaxings
    set GITHUB_REPO=url-navigator
)

echo [INFO] GitHub repository: %GITHUB_OWNER%/%GITHUB_REPO%
echo.

:: Confirm operation
echo The following operations will be performed:
echo - Update version from %CURRENT_VERSION% to %NEW_VERSION%
echo - Build Windows application
echo - Create Git tag and push
echo - Create GitHub Release
echo.
set /p CONFIRM="Confirm to continue? (y/N): "
if /i not "%CONFIRM%"=="y" (
    echo [INFO] Operation cancelled
    pause
    exit /b 0
)

echo.
echo [INFO] Starting release process...
echo.

:: Update version numbers
echo [INFO] Updating version numbers...

:: Backup original files
copy wails.json wails.json.backup >nul
if exist "frontend\package.json" copy frontend\package.json frontend\package.json.backup >nul

:: Update wails.json using PowerShell (store version WITHOUT v prefix in config files)
powershell -command "$json = Get-Content 'wails.json' | ConvertFrom-Json; $json.info.version = '%VERSION_NUMBER%'; $json.github.owner = '%GITHUB_OWNER%'; $json.github.repo = '%GITHUB_REPO%'; $json | ConvertTo-Json -Depth 10 | Set-Content 'wails.json'"

:: Update frontend/package.json (if exists)
if exist "frontend\package.json" (
    powershell -command "$json = Get-Content 'frontend\package.json' | ConvertFrom-Json; $json.version = '%VERSION_NUMBER%'; $json | ConvertTo-Json -Depth 10 | Set-Content 'frontend\package.json'"
)

echo [SUCCESS] Version numbers updated
echo.

:: Skip build check
if "%SKIP_BUILD%"=="true" (
    echo [INFO] Skipping build step
    goto commit_changes
)

:: Build application
echo [INFO] Starting application build...

:: Install frontend dependencies
echo [INFO] Installing frontend dependencies...
cd frontend
yarn install
if errorlevel 1 (
    echo [ERROR] Failed to install frontend dependencies
    cd ..
    pause
    exit /b 1
)
cd ..

:: Build frontend
echo [INFO] Building frontend...
cd frontend
yarn build
if errorlevel 1 (
    echo [ERROR] Frontend build failed
    cd ..
    pause
    exit /b 1
)
cd ..

:: Build Wails application
echo [INFO] Building Wails application...
wails build -platform windows/amd64 -ldflags "-H=windowsgui -s -w -X main.Version=%VERSION_NUMBER% -X main.GitHubOwner=%GITHUB_OWNER% -X main.GitHubRepo=%GITHUB_REPO%" -tags production -trimpath
if errorlevel 1 (
    echo [ERROR] Wails application build failed
    pause
    exit /b 1
)

:: Check build result
if not exist "build\bin\URLNavigator.exe" (
    echo [ERROR] Build file does not exist: build\bin\URLNavigator.exe
    pause
    exit /b 1
)

echo [SUCCESS] Application build completed successfully
dir "build\bin\URLNavigator.exe"
echo.

:commit_changes
:: Commit version update
echo [INFO] Committing version update...
git add wails.json
if exist "frontend\package.json" git add frontend\package.json
git commit -m "chore: bump version to %NEW_VERSION%

- Update version in wails.json
- Update frontend package.json
- Prepare for Windows release"

if errorlevel 1 (
    echo [ERROR] Commit failed
    pause
    exit /b 1
)

echo [SUCCESS] Version update committed
echo.

:: Skip release check
if "%SKIP_RELEASE%"=="true" (
    echo [INFO] Skipping release step
    goto finish
)

:: Create tag
echo [INFO] Creating tag %NEW_VERSION%...
git tag -a "%NEW_VERSION%" -m "Release %NEW_VERSION%

Features:
- Bookmark management with categories
- Auto-update functionality
- Modern UI with shadcn/ui
- Local data storage

Platform: Windows x64
Build: Automated release with version injection"

if errorlevel 1 (
    echo [ERROR] Failed to create tag
    pause
    exit /b 1
)

echo [SUCCESS] Tag created successfully
echo.

:: Push to remote repository
echo [INFO] Pushing to remote repository...

:: Try pushing to main branch first
git push origin main >nul 2>&1
if not errorlevel 1 (
    echo [SUCCESS] Code pushed to main branch
) else (
    :: Try pushing to master branch
    git push origin master >nul 2>&1
    if not errorlevel 1 (
        echo [SUCCESS] Code pushed to master branch
    ) else (
        echo [ERROR] Push failed. Please check remote repository configuration
        pause
        exit /b 1
    )
)

echo [INFO] Pushing tag...
git push origin "%NEW_VERSION%"
if errorlevel 1 (
    echo [ERROR] Tag push failed
    pause
    exit /b 1
)

echo [SUCCESS] Tag pushed successfully
echo.

:finish
:: Completion message
echo.
echo ======================================
echo [SUCCESS] Windows version %NEW_VERSION% release completed!
echo ======================================
echo.

echo What happens next:
echo   1. GitHub Actions will automatically start building Windows version
echo   2. A GitHub Release will be created automatically after build completion
echo   3. The Release will contain URLNavigator.exe file
echo   4. Application auto-update will detect the new version
echo.

echo Monitoring links:
echo   - Build status: https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/actions
echo   - Releases page: https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/releases
echo.

:: Wait and open browser
echo [INFO] Opening GitHub Actions page in 3 seconds...
timeout /t 3 /nobreak >nul
start "" "https://github.com/%GITHUB_OWNER%/%GITHUB_REPO%/actions"

echo.
echo [INFO] Release process completed! 🚀
echo.
pause