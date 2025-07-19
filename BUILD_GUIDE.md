# Build and Release Guide

## Quick Start

Use the automated Windows batch script for easy version management and release:

```bash
# Release a new version (e.g., v1.3.0)
release.bat v1.3.0

# Build only, skip release
release.bat v1.3.0 --skip-release

# Release only (requires existing build)
release.bat v1.3.0 --skip-build
```

## Version Format

- **Required format**: `vX.Y.Z` (e.g., `v1.3.0`)
- **Alternative**: `X.Y.Z` (script will add `v` prefix automatically)
- Examples: `v1.3.0`, `v2.0.1`, `v1.2.10`

## What the Script Does

1. **Environment Check**: Verifies Git, Go, Wails, and Yarn are installed
2. **Repository Validation**: Ensures clean working directory
3. **Version Management**: Updates version across all config files
4. **Build Process**: Compiles frontend and Windows executable
5. **Git Operations**: Creates tags and pushes to repository
6. **GitHub Integration**: Triggers automated GitHub Actions build

## Prerequisites

- Git (for version control)
- Go 1.21+ (for backend)
- Wails CLI v2.10+ (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
- Node.js 18+ and Yarn (for frontend)
- Clean Git working directory (no uncommitted changes)

## Manual Build (Development)

```bash
# Development mode
wails dev

# Build manually
cd frontend
yarn install
yarn build
cd ..
wails build
```

## Version Management

The application uses a multi-source version management system:

1. **Build-time injection**: Version injected via `-ldflags` during compilation
2. **wails.json**: Primary configuration file
3. **User config**: `%APPDATA%\.urlnavigator\version.json`
4. **Fallback**: Hardcoded defaults

## Automated Release Process

When you run `release.bat v1.3.0`:

1. Script updates `wails.json` and `frontend/package.json`
2. Commits version changes to Git
3. Creates and pushes a Git tag (e.g., `v1.3.0`)
4. GitHub Actions automatically:
   - Builds the Windows application
   - Creates a GitHub Release
   - Uploads `URLNavigator.exe` as release asset
5. Application auto-update detects the new version

## Monitoring

After running the script, monitor:
- **Build Status**: https://github.com/wangyaxings/url-navigator/actions
- **Releases**: https://github.com/wangyaxings/url-navigator/releases

## Troubleshooting

### Script Fails with "Tool not found"
- Install missing tools (Git, Go, Wails, Yarn)
- Ensure tools are in your system PATH

### "Working directory has uncommitted changes"
- Commit or stash your changes: `git add . && git commit -m "your message"`

### Build Fails
- Check if all dependencies are installed: `yarn install` in frontend directory
- Verify Wails setup: `wails doctor`

### GitHub Push Fails
- Verify Git remote configuration: `git remote -v`
- Check if you have push permissions to the repository

## Version Display

The application dynamically fetches version information from the backend:
- Frontend components show version with `v` prefix (e.g., `v1.3.0`)
- Configuration files store version without `v` prefix for compatibility
- Auto-update system handles both formats correctly

## Development Workflow

1. Make your changes
2. Test locally with `wails dev`
3. Commit changes: `git add . && git commit -m "feat: your feature"`
4. Run release script: `release.bat vX.Y.Z`
5. Monitor GitHub Actions for automated build and release