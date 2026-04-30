@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0package-cross.ps1" %*
exit /b %ERRORLEVEL%
