#Requires -Version 5.1

<#
.SYNOPSIS
    URL Navigator Windows Release Script

.DESCRIPTION
    Automated build, version management and release script for URL Navigator.
    Supports version format vX.Y.Z or X.Y.Z and manages all configuration through version.json.

.PARAMETER Version
    Target version (e.g., v1.3.0 or 1.3.0)

.PARAMETER SkipBuild
    Skip the build process, only perform release operations

.PARAMETER SkipRelease
    Skip the release process, only perform build operations

.PARAMETER Force
    Force execution without confirmation prompts

.EXAMPLE
    .\release.ps1 v1.3.0

.EXAMPLE
    .\release.ps1 1.3.0 -SkipBuild

.EXAMPLE
    .\release.ps1 v1.3.0 -Force
#>

param(
    [Parameter(Mandatory = $true, Position = 0)]
    [string]$Version,

    [switch]$SkipBuild,
    [switch]$SkipRelease,
    [switch]$Force
)

# Script configuration
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"

# Git retry configuration
$GitRetryConfig = @{
    CodePushRetries = 3      # 代码推送重试次数
    CodePushDelay = 5        # 代码推送延迟秒数
    TagPushRetries = 5       # 标签推送重试次数
    TagPushDelay = 8         # 标签推送延迟秒数
}

# Color functions for output
function Write-Info($Message) {
    Write-Host "[INFO] $Message" -ForegroundColor Cyan
}

function Write-Success($Message) {
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warning($Message) {
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Error($Message) {
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Header($Message) {
    Write-Host ""
    Write-Host "==========================================" -ForegroundColor Magenta
    Write-Host $Message -ForegroundColor Magenta
    Write-Host "==========================================" -ForegroundColor Magenta
    Write-Host ""
}

# Git 推送重试机制函数
function Invoke-GitPushWithRetry {
    param(
        [string]$Command,
        [string]$Description,
        [int]$MaxRetries = 3,
        [int]$DelaySeconds = 5
    )

    Write-Info $Description

    for ($attempt = 1; $attempt -le $MaxRetries; $attempt++) {
        try {
            Write-Info "Attempt $attempt/${MaxRetries}: $Command"

            # 执行 Git 命令
            $result = Invoke-Expression "$Command 2>&1"

            if ($LASTEXITCODE -eq 0) {
                Write-Success "✅ $Description completed successfully"
                return $true
            }
            else {
                $errorMsg = "Exit code: $LASTEXITCODE, Output: $result"

                if ($attempt -eq $MaxRetries) {
                    Write-Error "❌ $Description failed after ${MaxRetries} attempts"
                    Write-Error $errorMsg
                    throw "$Description failed: $errorMsg"
                }
                else {
                    Write-Warning "⚠️ Attempt $attempt failed: $errorMsg"
                    Write-Info "⏱️ Waiting $DelaySeconds seconds before retry..."
                    Start-Sleep -Seconds $DelaySeconds
                }
            }
        }
        catch {
            if ($attempt -eq $MaxRetries) {
                Write-Error "❌ $Description failed after ${MaxRetries} attempts"
                throw $_.Exception
            }
            else {
                Write-Warning "⚠️ Attempt $attempt failed: $($_.Exception.Message)"
                Write-Info "⏱️ Waiting $DelaySeconds seconds before retry..."
                Start-Sleep -Seconds $DelaySeconds
            }
        }
    }
}

# 增强的网络连接测试
function Test-GitHubConnectivity {
    Write-Info "🔍 Testing GitHub connectivity..."

    try {
        # 测试 HTTPS 连接
        $httpsTest = Test-NetConnection github.com -Port 443 -InformationLevel Quiet -WarningAction SilentlyContinue
        if ($httpsTest) {
            Write-Success "✅ GitHub HTTPS connection is working"
            return $true
        }
        else {
            Write-Warning "⚠️ GitHub HTTPS connection failed"
            return $false
        }
    }
    catch {
        Write-Warning "⚠️ Network connectivity test failed: $($_.Exception.Message)"
        return $false
    }
}

# Utility function to write formatted JSON without BOM
function Write-JsonWithoutBOM($Content, $FilePath) {
    # Format JSON with proper indentation for better readability
    try {
        # Parse and re-format JSON to ensure consistent formatting
        $jsonObject = $Content | ConvertFrom-Json
        $formattedJson = $jsonObject | ConvertTo-Json -Depth 10 -Compress:$false

        # Use UTF8 encoding without BOM for better compatibility with tools
        if ($PSVersionTable.PSVersion.Major -ge 6) {
            # PowerShell Core/7+ supports UTF8NoBOM
            $formattedJson | Set-Content $FilePath -Encoding UTF8NoBOM
        } else {
            # Windows PowerShell 5.1 - use .NET method to write without BOM
            $utf8NoBom = New-Object System.Text.UTF8Encoding $false
            [System.IO.File]::WriteAllText($FilePath, $formattedJson, $utf8NoBom)
        }

        # Verify the file was written correctly
        Confirm-JsonFileIntegrity -FilePath $FilePath
        Write-Info "✅ Written formatted JSON file without BOM: $FilePath"
    }
    catch {
        Write-Error "Failed to format JSON for $FilePath`: $($_.Exception.Message)"
        throw "JSON formatting failed"
    }
}

# Function to check and fix JSON file encoding issues
function Confirm-JsonFileIntegrity($FilePath) {
    try {
        # Check for BOM
        $bytes = [System.IO.File]::ReadAllBytes($FilePath)
        if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF) {
            Write-Warning "⚠️  BOM detected in $FilePath, this should not happen!"
            return $false
        }

        # Validate JSON syntax
        $content = Get-Content $FilePath -Raw
        $null = $content | ConvertFrom-Json
        Write-Info "✅ JSON integrity confirmed: $FilePath"
        return $true
    }
    catch {
        Write-Error "❌ JSON integrity check failed for $FilePath`: $($_.Exception.Message)"
        return $false
    }
}

# Function to auto-fix existing JSON files with BOM issues
function Repair-JsonFiles() {
    Write-Info "🔧 Checking and repairing JSON files..."

    $jsonFiles = @("wails.json", "version.json", "frontend/package.json")
    $repairedFiles = @()

    foreach ($file in $jsonFiles) {
        if (Test-Path $file) {
            $bytes = [System.IO.File]::ReadAllBytes($file)
            if ($bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF) {
                Write-Warning "🛠️  Repairing BOM in $file..."
                try {
                    $content = Get-Content $file -Raw | ConvertFrom-Json | ConvertTo-Json -Depth 10 -Compress:$false
                    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
                    [System.IO.File]::WriteAllText($file, $content, $utf8NoBom)
                    $repairedFiles += $file
                    Write-Success "✅ Repaired $file"
                }
                catch {
                    Write-Error "❌ Failed to repair $file`: $($_.Exception.Message)"
                }
            }
        }
    }

    if ($repairedFiles.Count -gt 0) {
        Write-Success "🔧 Repaired $($repairedFiles.Count) JSON files: $($repairedFiles -join ', ')"
    } else {
        Write-Info "✅ All JSON files are already in correct format"
    }
}

# Configuration and validation functions
function Initialize-Configuration {
    Write-Header "URL Navigator Windows Release Tool"

    # Validate PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Error "PowerShell 5.1 or higher is required"
        exit 1
    }

    # Check for required files
    $requiredFiles = @("version.json", "wails.json")
    foreach ($file in $requiredFiles) {
        if (-not (Test-Path $file)) {
            Write-Error "Required file not found: $file"
            exit 1
        }
    }

    # Validate and repair JSON files if needed
    Repair-JsonFiles

    # Load configuration from version.json (serves as template)
    try {
        $configContent = Get-Content "version.json" -Raw -Encoding UTF8
        $config = $configContent | ConvertFrom-Json
        Write-Success "Configuration loaded from version.json"
        return $config
    }
    catch {
        Write-Error "Failed to load configuration from version.json: $($_.Exception.Message)"
        exit 1
    }
}

function Test-Prerequisites {
    Write-Info "Checking prerequisites..."

    # Check Git
    try {
        $null = git --version 2>$null
        Write-Success "✅ Git is available"
    }
    catch {
        Write-Error "❌ Git is not installed or not in PATH"
        return $false
    }

    # Check Go
    try {
        $null = go version 2>$null
        Write-Success "✅ Go is available"
    }
    catch {
        Write-Error "❌ Go is not installed or not in PATH"
        return $false
    }

    # Check Wails
    try {
        $null = wails version 2>$null
        Write-Success "✅ Wails is available"
    }
    catch {
        Write-Error "❌ Wails is not installed. Install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        return $false
    }

    # Check Node.js and Yarn
    try {
        $null = node --version 2>$null
        Write-Success "✅ Node.js is available"
    }
    catch {
        Write-Error "❌ Node.js is not installed or not in PATH"
        return $false
    }

    try {
        $null = yarn --version 2>$null
        Write-Success "✅ Yarn is available"
    }
    catch {
        Write-Error "❌ Yarn is not installed or not in PATH"
        return $false
    }

    return $true
}

function Test-VersionFormat($Version) {
    # Support both vX.Y.Z and X.Y.Z formats
    if ($Version -match '^v?\d+\.\d+\.\d+$') {
        return $true
    }
    else {
        Write-Error "Invalid version format: $Version (expected: vX.Y.Z or X.Y.Z)"
        return $false
    }
}

function Get-NormalizedVersion($Version) {
    # Ensure version starts with 'v' and create both formats
    $versionWithoutV = $Version -replace '^v', ''
    return @{
        WithV = "v$versionWithoutV"
        WithoutV = $versionWithoutV
    }
}

function Test-GitRepository {
    Write-Info "Checking git repository status..."

    # Check if we're in a git repository
    try {
        $null = git rev-parse --git-dir 2>$null
        if ($LASTEXITCODE -ne 0) {
            Write-Error "Not in a git repository"
            return $false
        }
    }
    catch {
        Write-Error "Git repository check failed"
        return $false
    }

    # Check for uncommitted changes
    $status = git status --porcelain 2>$null
    if ($status) {
        Write-Error "Working directory is not clean. Please commit or stash your changes:"
        git status --short
        return $false
    }

    Write-Success "✅ Git repository is clean"
    return $true
}

function Get-CurrentVersion {
    Write-Info "Detecting current version..."

    # Try to get version from wails.json first
    if (Test-Path "wails.json") {
        try {
            $wailsConfig = Get-Content "wails.json" -Raw | ConvertFrom-Json
            if ($wailsConfig.info -and $wailsConfig.info.version) {
                Write-Success "Current version detected from wails.json: v$($wailsConfig.info.version)"
                return $wailsConfig.info.version
            }
        }
        catch {
            Write-Warning "Failed to read version from wails.json: $($_.Exception.Message)"
        }
    }

    # Try to get version from frontend/package.json
    if (Test-Path "frontend/package.json") {
        try {
            $packageConfig = Get-Content "frontend/package.json" -Raw | ConvertFrom-Json
            if ($packageConfig.version) {
                Write-Success "Current version detected from frontend/package.json: v$($packageConfig.version)"
                return $packageConfig.version
            }
        }
        catch {
            Write-Warning "Failed to read version from frontend/package.json"
        }
    }

    # Fallback to version.json (though this is meant to be a template)
    try {
        $versionConfig = Get-Content "version.json" -Raw | ConvertFrom-Json
        if ($versionConfig.version) {
            Write-Warning "Using version from version.json template: v$($versionConfig.version)"
            return $versionConfig.version
        }
    }
    catch {
        Write-Error "Failed to detect current version from any source"
        exit 1
    }

    Write-Error "Could not determine current version"
    exit 1
}

function Get-GitHubInfo($Config) {
    Write-Info "Detecting GitHub repository information..."

    # Try to auto-detect from git remote
    try {
        $remoteUrl = git config --get remote.origin.url 2>$null
        if ($remoteUrl -and $remoteUrl -match 'github\.com[:/]([^/]+)/([^/\.]+)(\.git)?/?$') {
            $owner = $Matches[1]
            $repo = $Matches[2]
            Write-Success "Auto-detected GitHub repository: $owner/$repo"
            return @{ Owner = $owner; Repo = $repo }
        }
    }
    catch {
        Write-Warning "Could not auto-detect GitHub repository from git remote"
    }

    # Use configuration values
    $owner = $Config.github.owner
    $repo = $Config.github.repo
    Write-Info "Using GitHub repository from configuration: $owner/$repo"

    return @{ Owner = $owner; Repo = $repo }
}

function Confirm-Operation($CurrentVersion, $NewVersion, $SkipBuild, $SkipRelease, $Force) {
    if ($Force) {
        Write-Info "Force mode enabled, skipping confirmation"
        return $true
    }

    Write-Host ""
    Write-Host "The following operations will be performed:" -ForegroundColor Yellow
    Write-Host "- Update version from v$CurrentVersion to $($NewVersion.WithV)" -ForegroundColor White

    if (-not $SkipBuild) {
        Write-Host "- Build Windows application" -ForegroundColor White
    }

    if (-not $SkipRelease) {
        Write-Host "- Create Git tag and push to repository" -ForegroundColor White
        Write-Host "- Trigger GitHub Actions build" -ForegroundColor White
    }

    Write-Host ""
    $confirmation = Read-Host "Confirm to continue? (y/N)"
    return $confirmation -eq 'y' -or $confirmation -eq 'Y'
}

function Update-VersionFiles($NewVersion, $GitHubInfo, $Config) {
    Write-Info "Updating version files..."
    Write-Info "Note: version.json will be preserved as configuration template"

    # Update wails.json
    if (Test-Path "wails.json") {
        try {
            $wailsConfig = Get-Content "wails.json" -Raw | ConvertFrom-Json

            # Ensure info section exists
            if (-not $wailsConfig.info) {
                $wailsConfig | Add-Member -NotePropertyName "info" -NotePropertyValue @{}
            }
            $wailsConfig.info.version = $NewVersion.WithoutV

            # Ensure github section exists and update repository info
            if (-not $wailsConfig.github) {
                $wailsConfig | Add-Member -NotePropertyName "github" -NotePropertyValue @{}
            }
            $wailsConfig.github.owner = $GitHubInfo.Owner
            $wailsConfig.github.repo = $GitHubInfo.Repo

            # Write JSON without BOM to avoid encoding issues
            $jsonContent = $wailsConfig | ConvertTo-Json -Depth 10
            Write-JsonWithoutBOM -Content $jsonContent -FilePath "wails.json"
            Write-Success "Updated wails.json to version $($NewVersion.WithoutV)"
        }
        catch {
            Write-Error "Failed to update wails.json: $($_.Exception.Message)"
            throw "wails.json update failed"
        }
    }

    # Update frontend/package.json if it exists
    if (Test-Path "frontend/package.json") {
        try {
            $packageConfig = Get-Content "frontend/package.json" -Raw | ConvertFrom-Json
            $packageConfig.version = $NewVersion.WithoutV

            # Write JSON without BOM to avoid encoding issues
            $jsonContent = $packageConfig | ConvertTo-Json -Depth 10
            Write-JsonWithoutBOM -Content $jsonContent -FilePath "frontend/package.json"
            Write-Success "Updated frontend/package.json to version $($NewVersion.WithoutV)"
        }
        catch {
            Write-Warning "Failed to update frontend/package.json: $($_.Exception.Message)"
        }
    }

    Write-Success "Version numbers updated successfully (version.json preserved as configuration template)"
}

function Build-Application($NewVersion, $GitHubInfo, $Config) {
    Write-Info "Starting application build..."

    # Install frontend dependencies
    Write-Info "Installing frontend dependencies..."
    Push-Location "frontend"
    try {
        $result = & yarn install --frozen-lockfile 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "yarn install failed: $result"
        }
        Write-Success "Frontend dependencies installed"
    }
    finally {
        Pop-Location
    }

    # Build frontend
    Write-Info "Building frontend..."
    Push-Location "frontend"
    try {
        $result = & yarn build 2>&1
        if ($LASTEXITCODE -ne 0) {
            throw "Frontend build failed: $result"
        }
        Write-Success "Frontend build completed"
    }
    finally {
        Pop-Location
    }

    # Build Wails application
    Write-Info "Building Wails application..."
    $ldflags = $Config.build.ldflags + @(
        "-X main.Version=$($NewVersion.WithoutV)",
        "-X main.GitHubOwner=$($GitHubInfo.Owner)",
        "-X main.GitHubRepo=$($GitHubInfo.Repo)"
    )

    # Combine ldflags into a single quoted string
    $ldflagsString = $ldflags -join " "

    $buildArgs = @(
        "build",
        "-platform", $Config.build.platform,
        "-ldflags", $ldflagsString
    ) + $Config.build.flags

    Write-Info "Build command: wails build -platform $($Config.build.platform) -ldflags `"$ldflagsString`" $($Config.build.flags -join ' ')"

    # Execute Wails build - use simpler approach to avoid PowerShell stderr issues
    Write-Info "Executing Wails build..."

    # Temporarily set ErrorActionPreference to Continue to avoid stderr issues
    $oldErrorAction = $ErrorActionPreference
    $ErrorActionPreference = "Continue"

    try {
        # Execute wails command and capture only exit code
        & wails @buildArgs
        $buildExitCode = $LASTEXITCODE

        # Restore original error action
        $ErrorActionPreference = $oldErrorAction

        if ($buildExitCode -ne 0) {
            Write-Error "Wails build failed with exit code: $buildExitCode"
            throw "Build failed"
        }

        Write-Success "Wails build completed successfully (exit code: 0)"
    }
    catch {
        $ErrorActionPreference = $oldErrorAction
        Write-Error "Failed to execute Wails build: $($_.Exception.Message)"
        throw "Build execution failed"
    }

    # Check build result
    $exePath = "build/bin/URLNavigator.exe"
    if (-not (Test-Path $exePath)) {
        throw "Build file does not exist: $exePath"
    }

    $fileInfo = Get-Item $exePath
    Write-Success "Application build completed successfully"
    Write-Info "Build size: $([math]::Round($fileInfo.Length / 1MB, 2)) MB ($($fileInfo.Length) bytes)"
}

# 修改后的 Invoke-GitOperations 函数（带重试机制）
function Invoke-GitOperations($NewVersion, $Config) {
    Write-Info "Committing version update..."

    # Add modified files (excluding version.json which remains as config template)
    Write-Info "Adding wails.json and frontend/package.json to git..."
    git add wails.json 2>$null
    git add frontend/package.json 2>$null

    # Check if there are any changes to commit
    $status = git status --porcelain 2>$null
    if (-not $status) {
        Write-Warning "No changes to commit. Files may already be up to date."
        return
    }

    # Create commit message
    $commitMessage = $Config.release.commit_message_template -replace '\{version\}', $NewVersion.WithV
    $commitMessage += "`n`n- Update version in wails.json to $($NewVersion.WithoutV)`n- Update version in frontend/package.json to $($NewVersion.WithoutV)`n- Prepare for Windows release $($NewVersion.WithV)`n- Keep version.json as stable configuration template"

    $result = git commit -m $commitMessage 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Git commit failed: $result"
        throw "Commit failed"
    }

    Write-Success "Version update committed (version.json preserved)"

    # Create tag
    Write-Info "Creating tag $($NewVersion.WithV)..."
    $tagMessage = $Config.release.tag_message_template -replace '\{version\}', $NewVersion.WithV

    $result = git tag -a $NewVersion.WithV -m $tagMessage 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create tag: $result"
        throw "Tag creation failed"
    }

    Write-Success "Tag created successfully"

    # Test GitHub connectivity before pushing
    $connectivityOk = Test-GitHubConnectivity
    if (-not $connectivityOk) {
        Write-Warning "⚠️ GitHub connectivity issues detected, but continuing with retry mechanism..."
    }

    # Get current branch
    $currentBranch = git branch --show-current 2>$null
    if (-not $currentBranch) {
        $currentBranch = "main"
        Write-Warning "Cannot detect current branch, defaulting to 'main'"
    }

    # Push to remote repository with retry mechanism
    try {
        # 推送代码到分支（带重试）
        Invoke-GitPushWithRetry -Command "git push origin $currentBranch" -Description "Pushing code to $currentBranch branch" -MaxRetries $GitRetryConfig.CodePushRetries -DelaySeconds $GitRetryConfig.CodePushDelay

        Write-Success "Code pushed to $currentBranch branch"

        # 推送标签（带重试，增加延迟因为标签推送通常需要更多时间）
        Write-Info "🏷️ Pushing tag $($NewVersion.WithV) to trigger GitHub Actions..."
        Invoke-GitPushWithRetry -Command "git push origin $($NewVersion.WithV)" -Description "Pushing tag $($NewVersion.WithV)" -MaxRetries $GitRetryConfig.TagPushRetries -DelaySeconds $GitRetryConfig.TagPushDelay

        Write-Success "✅ Tag $($NewVersion.WithV) pushed successfully!"
        Write-Success "🚀 GitHub Actions should now be triggered for release creation"

        # Verify tag exists on remote with retry
        Write-Info "🔍 Verifying tag on remote repository..."
        $verificationSuccess = $false

        for ($i = 1; $i -le 3; $i++) {
            Start-Sleep -Seconds 2
            $remoteTag = git ls-remote --tags origin $NewVersion.WithV 2>$null
            if ($remoteTag) {
                Write-Success "✅ Tag verified on remote: $($NewVersion.WithV)"
                $verificationSuccess = $true
                break
            }
            else {
                Write-Info "⏱️ Verification attempt $i/3: Tag not yet visible on remote..."
            }
        }

        if (-not $verificationSuccess) {
            Write-Warning "⚠️ Tag verification failed, but push seemed successful. GitHub Actions may still trigger."
        }
    }
    catch {
        Write-Error "❌ Git operations failed: $($_.Exception.Message)"
        Write-Info ""
        Write-Info "💡 Manual recovery options:"
        Write-Info "   1. Run: git push origin $currentBranch"
        Write-Info "   2. Run: git push origin $($NewVersion.WithV)"
        Write-Info "   3. Check GitHub repository for partial updates"
        Write-Info ""
        throw "Git push operations failed"
    }
}

function Show-CompletionMessage($NewVersion, $GitHubInfo, $Config) {
    Write-Host ""
    Write-Header "Windows version $($NewVersion.WithV) release completed! 🎉"

    Write-Host "What happens next:" -ForegroundColor Yellow
    Write-Host "  1. 🔄 GitHub Actions will automatically start building Windows version" -ForegroundColor White
    Write-Host "  2. 📦 A GitHub Release will be created automatically after build completion" -ForegroundColor White
    Write-Host "  3. 💾 The Release will contain URLNavigator.exe file" -ForegroundColor White
    Write-Host "  4. 🔄 Application auto-update will detect the new version" -ForegroundColor White
    Write-Host ""

    Write-Host "Monitoring links:" -ForegroundColor Yellow
    $actionsUrl = "https://github.com/$($GitHubInfo.Owner)/$($GitHubInfo.Repo)/actions"
    $releasesUrl = "https://github.com/$($GitHubInfo.Owner)/$($GitHubInfo.Repo)/releases"

    Write-Host "  - 🔍 Build status: $actionsUrl" -ForegroundColor Cyan
    Write-Host "  - 📋 Releases page: $releasesUrl" -ForegroundColor Cyan
    Write-Host ""

    if ($Config.release.auto_open_browser) {
        Write-Info "Opening GitHub Actions page in 3 seconds..."
        Start-Sleep -Seconds 3
        Start-Process $actionsUrl
    }

    Write-Success "Release process completed! 🚀"
}

# Main execution function
function Main {
    try {
        # Initialize and validate environment
        $config = Initialize-Configuration

        # Check prerequisites
        if (-not (Test-Prerequisites)) {
            Write-Error "Prerequisites check failed"
            exit 1
        }

        # Validate version format
        if (-not (Test-VersionFormat $Version)) {
            exit 1
        }

        # Normalize version format
        $newVersion = Get-NormalizedVersion $Version
        Write-Success "Target version: $($newVersion.WithV)"
        Write-Info "Version number: $($newVersion.WithoutV)"
        Write-Info "Skip build: $SkipBuild"
        Write-Info "Skip release: $SkipRelease"

        # Check git repository status
        if (-not (Test-GitRepository)) {
            exit 1
        }

        # Get current version and GitHub info
        $currentVersion = Get-CurrentVersion
        $gitHubInfo = Get-GitHubInfo $config

        Write-Info "Current version: v$currentVersion"
        Write-Info "GitHub repository: $($gitHubInfo.Owner)/$($gitHubInfo.Repo)"

        # Check if version is different
        if ($newVersion.WithoutV -eq $currentVersion) {
            Write-Error "New version cannot be the same as current version"
            exit 1
        }

        # Confirm operation
        if (-not (Confirm-Operation $currentVersion $newVersion $SkipBuild $SkipRelease $Force)) {
            Write-Info "Operation cancelled"
            exit 0
        }

        # Update version files
        Update-VersionFiles $newVersion $gitHubInfo $config

        # Verify all JSON files after update
        Write-Info "🔍 Verifying JSON files integrity after update..."
        $integrity = @()
        $integrity += Confirm-JsonFileIntegrity "wails.json"
        $integrity += Confirm-JsonFileIntegrity "version.json"
        if (Test-Path "frontend/package.json") {
            $integrity += Confirm-JsonFileIntegrity "frontend/package.json"
        }

        if ($integrity -contains $false) {
            Write-Error "❌ JSON integrity check failed, aborting release"
            throw "JSON integrity validation failed"
        }
        Write-Success "✅ All JSON files passed integrity check"

        # Build application (if not skipped)
        if (-not $SkipBuild) {
            Build-Application $newVersion $gitHubInfo $config
        }
        else {
            Write-Info "Skipping build step"
        }

        # Release operations (if not skipped)
        if (-not $SkipRelease) {
            Invoke-GitOperations $newVersion $config
        }
        else {
            Write-Info "Skipping release step"
        }

        # Show completion message
        Show-CompletionMessage $newVersion $gitHubInfo $config

    }
    catch {
        Write-Error "Script failed: $($_.Exception.Message)"
        Write-Error "Stack trace: $($_.ScriptStackTrace)"
        exit 1
    }
}

# Script entry point
Main