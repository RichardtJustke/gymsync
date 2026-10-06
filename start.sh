#!/bin/bash
# Script para iniciar backend e frontend do GymSync

set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
cd "$DIR"

echo "🏋️ Iniciando GymSync Ecosystem..."

# 1. Checa Docker para PostgreSQL (opcional, pois o backend possui fallback para SQLite local)
if command -v docker &> /dev/null && docker info &> /dev/null; then
  echo "🐘 Subindo banco de dados PostgreSQL via Docker..."
  docker compose up -d postgres || true
fi

# 2. Inicia Backend Go em segundo plano
echo "🚀 Iniciando Backend Golang na porta 8080..."
cd "$DIR/backend"
go run cmd/api/main.go &
BACKEND_PID=$!

# Função para encerrar ao sair (Ctrl+C)
trap "kill $BACKEND_PID 2>/dev/null || true; exit" SIGINT SIGTERM EXIT

# 3. Inicia Frontend Next.js na porta 3000
echo "🌐 Iniciando Frontend Next.js na porta 3000..."
cd "$DIR/frontend"
npm run dev

