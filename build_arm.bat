@echo off
setlocal
cd /d "%~dp0"

set CGO_ENABLED=0
set GOOS=linux
set GOARCH=arm
set GOARM=7

echo Compilando clpMonitor (linux/arm v7)...
go build -ldflags="-s -w" -o clpMonitor cmd/server/main.go

if errorlevel 1 (
    echo.
    echo [ERRO] Falha na compilacao.
) else (
    echo.
    echo [OK] Binario gerado: %~dp0clpMonitor
    echo Lembre-se de copiar tambem a pasta "web" para o dispositivo.
)

echo.
pause
endlocal

