@echo off
set DEST_DIR=c:\TOOLS\exe

:: Make sure the destination folder exists, if not, create it first
if not exist "%DEST_DIR%" mkdir "%DEST_DIR%"

echo Building...
go build -o mks.exe mks\main.go

if not %errorlevel%==0 (
    echo.
    echo [ERROR] BUILD FAILURE! Please improve your code.
    echo.
    :: pause
    exit /b %errorlevel%
) else (
    echo.
    echo Build Success! Copying files...
    copy /y mks.exe "%DEST_DIR%"
    echo.
    echo [COMPLETE] Files successfully copied to %DEST_DIR%
)

:: Delete the exe file in the source folder if you want to clean it (optional)
:: del mks.exe