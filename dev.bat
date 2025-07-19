@echo off
echo Starting URL Navigator in development mode...

echo Installing dependencies...
cd frontend
call yarn install
cd ..
go mod tidy

echo Starting Wails development server...
wails dev