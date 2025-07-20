@echo off
setlocal enabledelayedexpansion

echo ==========================================
echo URL Navigator Go Release Tool
echo ==========================================

if "%~1"=="" (
    echo Usage: release.bat ^<version^> [options]
    echo Options:
    echo   -skip-build    Skip build process
    echo   -skip-release  Skip release process
    echo   -force         Force execution, skip confirmation
    echo.
    echo Examples:
    echo   release.bat v1.3.0
    echo   release.bat 1.3.0 -skip-build
    echo   release.bat v1.3.0 -force
    goto :end
)

:: Run the Go release tool with all arguments
go run tools/release.go %*

:end
pause