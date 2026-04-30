@echo off
setlocal
chcp 65001 >nul

set "TOOLS_DIR=%~dp0"
set "APP_DIR=%TOOLS_DIR%.."
set "EXE_PATH=%APP_DIR%\Detour2.exe"
set "PS_SCRIPT=%TOOLS_DIR%Sign-WindowsLocal.ps1"

echo Detour2 本机测试签名工具
echo.

if not exist "%PS_SCRIPT%" (
    echo [错误] 找不到签名脚本：%PS_SCRIPT%
    echo.
    pause
    exit /b 1
)

if not exist "%EXE_PATH%" (
    echo [错误] 找不到 Detour2.exe。
    echo 请确认目录结构如下：
    echo   Detour2.exe
    echo   tools\Sign-WindowsLocal.cmd
    echo   tools\Sign-WindowsLocal.ps1
    echo.
    pause
    exit /b 1
)

echo 将要签名：%EXE_PATH%
echo.
echo 注意：这只会创建本机 self-signed 测试签名，不是正式发行签名。
echo.

powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%PS_SCRIPT%" -ExePath "%EXE_PATH%"
set "RC=%ERRORLEVEL%"

echo.
if "%RC%"=="0" (
    echo [完成] Detour2.exe 已完成本机测试签名。
) else (
    echo [失败] 签名失败，退出码：%RC%
)

echo.
pause
exit /b %RC%
