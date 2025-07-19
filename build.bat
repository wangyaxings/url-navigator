@echo off
echo Building URL Navigator...

echo Installing frontend dependencies...
cd frontend
call yarn install
if %errorlevel% neq 0 (
    echo Frontend dependency installation failed!
    exit /b %errorlevel%
)

echo Building frontend...
call yarn build
if %errorlevel% neq 0 (
    echo Frontend build failed!
    exit /b %errorlevel%
)

cd ..

echo Installing Go dependencies...
go mod tidy
if %errorlevel% neq 0 (
    echo Go dependency installation failed!
    exit /b %errorlevel%
)

echo Building application...
wails build
if %errorlevel% neq 0 (
    echo Wails build failed!
    exit /b %errorlevel%
)

echo Build completed successfully!
echo Executable location: ./build/bin/URLNavigator.exe