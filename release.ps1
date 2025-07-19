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

# Configuration and validation functions
function Initialize-Configuration {
    Write-Header "URL Navigator Windows Release Tool"

    # Validate PowerShell version
    if ($PSVersionTable.PSVersion.Major -lt 5) {
        Write-Error "PowerShell 5.1 or higher is required"
        exit 1
    }

    # Validate we're in a git repository
    if (-not (Test-Path ".git")) {
        Write-Error "Current directory is not a Git repository"
        exit 1
    }

    # Check for required tools
    Test-Prerequisites

    # Load or create version configuration
    return Get-VersionConfiguration
}

function Test-Prerequisites {
    Write-Info "Checking required tools..."

    $tools = @(
        @{Name = "git"; Command = "git --version"},
        @{Name = "go"; Command = "go version"},
        @{Name = "wails"; Command = "wails version"},
        @{Name = "yarn"; Command = "yarn --version"}
    )

    $missing = @()
    foreach ($tool in $tools) {
        try {
            $null = Invoke-Expression $tool.Command 2>$null
            Write-Host "  ✓ $($tool.Name)" -ForegroundColor Green
        }
        catch {
            Write-Host "  ✗ $($tool.Name)" -ForegroundColor Red
            $missing += $tool.Name
        }
    }

    if ($missing.Count -gt 0) {
        Write-Error "Missing required tools: $($missing -join ', ')"
        Write-Info "Please install missing tools and ensure they are in your PATH"
        exit 1
    }

    Write-Success "All required tools are installed"
}

function Get-VersionConfiguration {
    $versionFile = "version.json"

    if (-not (Test-Path $versionFile)) {
        Write-Info "Creating default version.json configuration..."
        $defaultConfig = @{
            version = "1.2.1"
            github = @{
                owner = "wangyaxings"
                repo = "url-navigator"
            }
            app = @{
                name = "URLNavigator"
                display_name = "URL Navigator"
                description = "A beautiful URL bookmark manager with auto-update functionality"
            }
            build = @{
                platform = "windows/amd64"
                flags = @("-tags", "production", "-trimpath", "-clean")
                ldflags = @("-H=windowsgui", "-s", "-w")
            }
            release = @{
                create_github_release = $true
                auto_open_browser = $true
                commit_message_template = "chore: bump version to {version}"
                tag_message_template = @"
Release {version}

🚀 Windows Release {version}

Features:
- Bookmark management with categories
- Auto-update functionality
- Modern UI with shadcn/ui
- Local data storage
- Improved performance and stability

Platform: Windows x64
Build: Automated release with version injection
"@
            }
        }

        $defaultConfig | ConvertTo-Json -Depth 10 | Set-Content $versionFile -Encoding UTF8
        Write-Success "Created $versionFile with default configuration"
    }

    try {
        $config = Get-Content $versionFile -Raw | ConvertFrom-Json
        Write-Success "Loaded configuration from $versionFile"
        return $config
    }
    catch {
        Write-Error "Failed to parse $versionFile`: $($_.Exception.Message)"
        exit 1
    }
}

function Test-VersionFormat($Version) {
    Write-Info "Validating version format..."

    if ($Version -match '^v?\d+\.\d+\.\d+$') {
        Write-Success "Version format validation passed"
        return $true
    }
    else {
        Write-Error "Invalid version format: '$Version'"
        Write-Error "Expected format: vX.Y.Z or X.Y.Z (e.g., v1.3.0 or 1.3.0)"
        return $false
    }
}

function Get-NormalizedVersion($Version) {
    if ($Version.StartsWith('v')) {
        $versionWithV = $Version
        $versionWithoutV = $Version.Substring(1)
    }
    else {
        $versionWithV = "v$Version"
        $versionWithoutV = $Version
    }

    return @{
        WithV = $versionWithV
        WithoutV = $versionWithoutV
    }
}

function Test-GitRepository {
    Write-Info "Checking Git repository status..."

    # Check if working directory is clean
    $status = git status --porcelain 2>$null
    if ($status) {
        Write-Error "Working directory has uncommitted changes:"
        git status --short
        Write-Info "Please commit or stash changes before releasing"
        return $false
    }

    Write-Success "Git repository status is clean"
    return $true
}

function Get-CurrentVersion($Config) {
    Write-Info "Getting current version information..."

    # Try to read from wails.json first
    if (Test-Path "wails.json") {
        try {
            $wailsConfig = Get-Content "wails.json" -Raw | ConvertFrom-Json
            $currentVersion = $wailsConfig.info.version
            if ($currentVersion) {
                Write-Info "Current version from wails.json: v$currentVersion"
                return $currentVersion
            }
        }
        catch {
            Write-Warning "Failed to read version from wails.json: $($_.Exception.Message)"
        }
    }

    # Fallback to version.json
    $currentVersion = $Config.version
    Write-Info "Current version from version.json: v$currentVersion"
    return $currentVersion
}

function Get-GitHubInfo($Config) {
    Write-Info "Getting GitHub repository information..."

    # Try to auto-detect from git remote
    try {
        $remoteUrl = git config --get remote.origin.url 2>$null
        if ($remoteUrl -and $remoteUrl -match 'github\.com[:/]([^/]+)/([^/]+?)(?:\.git)?/?$') {
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

    return $confirmation.ToLower() -eq 'y'
}

# Build and release functions
function Update-VersionFiles($NewVersion, $GitHubInfo, $Config) {
    Write-Info "Updating version numbers..."

    # Backup original files
    if (Test-Path "wails.json") {
        Copy-Item "wails.json" "wails.json.backup" -Force
    }
    if (Test-Path "frontend/package.json") {
        Copy-Item "frontend/package.json" "frontend/package.json.backup" -Force
    }

    # Update version.json
    $Config.version = $NewVersion.WithoutV
    $Config.github.owner = $GitHubInfo.Owner
    $Config.github.repo = $GitHubInfo.Repo
    $Config | ConvertTo-Json -Depth 10 | Set-Content "version.json" -Encoding UTF8

    # Update wails.json
    if (Test-Path "wails.json") {
        try {
            $wailsConfig = Get-Content "wails.json" -Raw | ConvertFrom-Json
            $wailsConfig.info.version = $NewVersion.WithoutV

            # Ensure github section exists
            if (-not $wailsConfig.github) {
                $wailsConfig | Add-Member -NotePropertyName "github" -NotePropertyValue @{}
            }
            $wailsConfig.github.owner = $GitHubInfo.Owner
            $wailsConfig.github.repo = $GitHubInfo.Repo

            $wailsConfig | ConvertTo-Json -Depth 10 | Set-Content "wails.json" -Encoding UTF8
            Write-Success "Updated wails.json"
        }
        catch {
            Write-Error "Failed to update wails.json: $($_.Exception.Message)"
            throw
        }
    }

    # Update frontend/package.json
    if (Test-Path "frontend/package.json") {
        try {
            $packageConfig = Get-Content "frontend/package.json" -Raw | ConvertFrom-Json
            $packageConfig.version = $NewVersion.WithoutV
            $packageConfig | ConvertTo-Json -Depth 10 | Set-Content "frontend/package.json" -Encoding UTF8
            Write-Success "Updated frontend/package.json"
        }
        catch {
            Write-Warning "Failed to update frontend/package.json: $($_.Exception.Message)"
        }
    }

    Write-Success "Version numbers updated successfully"
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

    $buildArgs = @(
        "build",
        "-platform", $Config.build.platform,
        "-ldflags", ($ldflags -join " ")
    ) + $Config.build.flags

    Write-Info "Build command: wails $($buildArgs -join ' ')"

    $result = & wails @buildArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Wails application build failed: $result"
        throw "Build failed"
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

function Invoke-GitOperations($NewVersion, $Config) {
    Write-Info "Committing version update..."

    # Add modified files
    git add version.json
    git add wails.json 2>$null
    git add frontend/package.json 2>$null

    # Create commit message
    $commitMessage = $Config.release.commit_message_template -replace '\{version\}', $NewVersion.WithV
    $commitMessage += "`n`n- Update version in configuration files to $($NewVersion.WithoutV)`n- Prepare for Windows release $($NewVersion.WithV)"

    $result = git commit -m $commitMessage 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Git commit failed: $result"
        throw "Commit failed"
    }

    Write-Success "Version update committed"

    # Create tag
    Write-Info "Creating tag $($NewVersion.WithV)..."
    $tagMessage = $Config.release.tag_message_template -replace '\{version\}', $NewVersion.WithV

    $result = git tag -a $NewVersion.WithV -m $tagMessage 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to create tag: $result"
        throw "Tag creation failed"
    }

    Write-Success "Tag created successfully"

    # Push to remote repository
    Write-Info "Pushing to remote repository..."

    # Get current branch
    $currentBranch = git branch --show-current 2>$null
    if (-not $currentBranch) {
        $currentBranch = "main"
        Write-Warning "Cannot detect current branch, defaulting to 'main'"
    }

    Write-Info "Pushing to branch: $currentBranch"
    $result = git push origin $currentBranch 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push code: $result"
        throw "Code push failed"
    }

    Write-Success "Code pushed to $currentBranch branch"

    # Push tag
    Write-Info "Pushing tag..."
    $result = git push origin $NewVersion.WithV 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to push tag: $result"
        throw "Tag push failed"
    }

    Write-Success "Tag pushed successfully"
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
        $currentVersion = Get-CurrentVersion $config
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