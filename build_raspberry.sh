#!/bin/bash
# Compila o clpMonitor para Raspberry Pi 3 B v1.2 (Raspberry Pi OS 32 bits = linux/arm v7).
# Para Raspberry Pi OS 64 bits, rode: ARCH=arm64 ./build_raspberry.sh
cd "$(dirname "$(readlink -f "$0")")" || exit 1

# Garante o Go no PATH mesmo quando aberto por duplo clique
export PATH="$HOME/.local/go/bin:/usr/local/go/bin:$PATH"

export CGO_ENABLED=0
export GOOS=linux
if [ "${ARCH:-arm}" = "arm64" ]; then
    export GOARCH=arm64
else
    export GOARCH=arm
    export GOARM=7
fi

echo "Compilando clpMonitor (linux/$GOARCH)..."
if go build -ldflags="-s -w" -o clpMonitor cmd/server/main.go; then
    echo
    echo "[OK] Binario gerado: $(pwd)/clpMonitor"
    echo "Copie tambem a pasta \"web\" e gateway_config.json para o Raspberry."
else
    echo
    echo "[ERRO] Falha na compilacao."
fi

echo
read -r -p "Pressione Enter para fechar..."
