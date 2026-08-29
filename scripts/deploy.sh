#!/bin/bash
# ==========================================
# ToenToen API - Server-side Deployment Script
# ==========================================
set -e

echo "🚀 Menjalankan deployment cepat di AWS EC2..."

# 1. Pastikan di direktori aplikasi
cd "$(dirname "$0")/.."

# 2. Tarik update repository
echo "📥 Git pull origin main..."
git pull origin main

# 3. Pull image pre-built dari GHCR (Tanpa beban build lokal)
echo "📦 Pulling api-server image dari GHCR..."
docker compose pull api-server

# 4. Jalankan migrasi database
echo "🗄️ Running database migrations..."
docker compose run --rm migrator || true

# 5. Restart container secara instan
echo "♻️ Starting up api-server..."
docker compose up -d --no-build api-server

# 6. Bersihkan image gantung
echo "🧹 Pruning old unused images..."
docker image prune -f

echo "🎉 Deployment selesai dengan sukses!"
