# Build and Release Guide

## Quick Start

Use the automated PowerShell script for easy version management and release:

```powershell
# Release a new version (e.g., v1.3.0)
.\release.ps1 v1.3.0

# Build only, skip release
.\release.ps1 v1.3.0 -SkipRelease

# Release only (requires existing build)
.\release.ps1 v1.3.0 -SkipBuild

# Force execution without confirmation
.\release.ps1 v1.3.0 -Force
```

## Version Format

- **Primary format**: `vX.Y.Z` (e.g., `v1.3.0`)
- **Alternative**: `X.Y.Z` (script will add `v` prefix automatically)
- Examples: `v1.3.0`, `v2.0.1`, `v1.2.10`

## Configuration Management

All configuration is now managed through `version.json`:

```json
{
  "version": "1.2.1",
  "github": {
    "owner": "wangyaxings",
    "repo": "url-navigator"
  },
  "app": {
    "name": "URLNavigator",
    "display_name": "URL Navigator",
    "description": "A beautiful URL bookmark manager"
  },
  "build": {
    "platform": "windows/amd64",
    "flags": ["-tags", "production", "-trimpath", "-clean"],
    "ldflags": ["-H=windowsgui", "-s", "-w"]
  },
  "release": {
    "create_github_release": true,
    "auto_open_browser": true,
    "commit_message_template": "chore: bump version to {version}",
    "tag_message_template": "Release {version}..."
  }
}
```

## What the Script Does

1. **Environment Check**: Verifies PowerShell 5.1+, Git, Go, Wails, and Yarn
2. **Configuration Loading**: Loads settings from `version.json`
3. **Version Validation**: Ensures proper version format
4. **Repository Validation**: Ensures clean working directory
5. **Auto-Detection**: Automatically detects GitHub repository from git remote
6. **Version Management**: Updates version across all config files
7. **Build Process**: Compiles frontend and Windows executable with injected version info
8. **Git Operations**: Creates tags and pushes to repository
9. **GitHub Integration**: Triggers automated GitHub Actions build

## Prerequisites

- **PowerShell 5.1+** (Windows PowerShell or PowerShell Core)
- **Git** (for version control)
- **Go 1.21+** (for backend)
- **Wails CLI v2.10+** (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- **Node.js 18+ and Yarn** (for frontend)
- **Clean Git working directory** (no uncommitted changes)

## Command-Line Options

```powershell
# Basic usage
.\release.ps1 <version>

# Available switches
-SkipBuild      # Skip the build process
-SkipRelease    # Skip the release process
-Force          # Skip confirmation prompts

# Examples
.\release.ps1 v1.3.0
.\release.ps1 1.3.0 -SkipBuild
.\release.ps1 v1.3.0 -Force
.\release.ps1 v1.3.0 -SkipBuild -SkipRelease  # Only update configs
```

## Manual Build (Development)

```powershell
# Development mode
wails dev

# Build manually
Set-Location frontend
yarn install
yarn build
Set-Location ..
wails build
```

## Automated Release Process

When you run `.\release.ps1 v1.3.0`:

1. **Validation**: Checks environment and version format
2. **Configuration**: Loads settings from `version.json`
3. **Auto-Detection**: Detects GitHub repository from git remote
4. **Updates**: Modifies `version.json`, `wails.json`, and `frontend/package.json`
5. **Build**: Compiles application with version injection
6. **Git Operations**: Commits, tags, and pushes changes
7. **GitHub Actions**: Automatically triggered for release creation

## Error Handling and Recovery

The PowerShell script includes comprehensive error handling:

- **Automatic Backups**: Creates `.backup` files before modifications
- **Rollback Support**: Detailed error messages for troubleshooting
- **Environment Validation**: Checks all prerequisites before execution
- **Git Safety**: Validates clean working directory
- **Build Verification**: Confirms successful compilation

## Configuration Customization

### GitHub Repository

The script auto-detects your repository from `git remote origin`, but you can override in `version.json`:

```json
{
  "github": {
    "owner": "your-username",
    "repo": "your-repo-name"
  }
}
```

### Build Settings

Customize build flags and target platform:

```json
{
  "build": {
    "platform": "windows/amd64",
    "flags": ["-tags", "production", "-trimpath"],
    "ldflags": ["-H=windowsgui", "-s", "-w"]
  }
}
```

### Release Behavior

Control release automation:

```json
{
  "release": {
    "create_github_release": true,
    "auto_open_browser": false,
    "commit_message_template": "feat: release {version}",
    "tag_message_template": "Custom release message..."
  }
}
```

## Monitoring and Links

After running the script:
- **Build Status**: https://github.com/wangyaxings/url-navigator/actions
- **Releases**: https://github.com/wangyaxings/url-navigator/releases

## Troubleshooting

### PowerShell Execution Policy

If you get execution policy errors:
```powershell
# Check current policy
Get-ExecutionPolicy

# Allow local scripts (run as Administrator)
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope LocalMachine

# Or run with bypass for single execution
powershell -ExecutionPolicy Bypass -File .\release.ps1 v1.3.0
```

### Script Fails with "Tool not found"
- Install missing tools (Git, Go, Wails, Yarn)
- Ensure tools are in your system PATH
- Restart PowerShell after PATH changes

### "Working directory has uncommitted changes"
```powershell
# Commit or stash your changes
git add .
git commit -m "your message"
# or
git stash
```

### Build Fails
```powershell
# Check dependencies
Set-Location frontend
yarn install
Set-Location ..

# Verify Wails setup
wails doctor

# Check Go modules
go mod tidy
```

### GitHub Push Fails
- Verify Git remote configuration: `git remote -v`
- Check if you have push permissions to the repository
- Ensure GitHub authentication is configured

## Version Display

The application dynamically fetches version information:
- Frontend components show version with `v` prefix (e.g., `v1.3.0`)
- Configuration files store version without `v` prefix for compatibility
- Auto-update system handles both formats correctly

## Development Workflow

1. **Make Changes**: Develop your features
2. **Test Locally**: `wails dev` for testing
3. **Commit Changes**: `git add . && git commit -m "feat: your feature"`
4. **Run Release**: `.\release.ps1 vX.Y.Z`
5. **Monitor Build**: Check GitHub Actions for automated build and release

## Migration from Batch Script

If migrating from `release.bat`:

1. **Keep both scripts** during transition period
2. **Test PowerShell script** with `-SkipRelease` first
3. **Verify configuration** in `version.json`
4. **Remove old script** after successful validation

The PowerShell script provides better:
- Error handling and recovery
- Configuration management
- Cross-platform compatibility (Windows PowerShell / PowerShell Core)
- Debugging and verbose output