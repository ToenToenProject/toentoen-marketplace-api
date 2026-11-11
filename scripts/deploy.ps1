param(
    [Parameter(Mandatory=$true)]
    [ValidateSet("dev", "staging", "preprod", "prod")]
    [string]$Target
)

$ErrorActionPreference = "Stop"

$EC2_HOST = "3.105.120.199"
$EC2_KEY = "D:\ToenToen\toentoen-key-ec2.pem"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  ToenToen API Deployment ke: $Target" -ForegroundColor Yellow
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Compile static Linux binary locally
Write-Host "[1/4] Meng-compile binary Linux x64 di lokal..." -ForegroundColor Green
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -ldflags="-s -w" -o server_linux ./cmd/api
if (-not (Test-Path "server_linux")) {
    Write-Error "Kompilasi gagal! File server_linux tidak ditemukan."
}

# 2. Upload via SCP
Write-Host "[2/4] Mengunggah binary ke EC2 via SCP..." -ForegroundColor Green
scp -i "$EC2_KEY" server_linux "ubuntu@$EC2_HOST`:/home/ubuntu/server_deploy"
Remove-Item -Force "server_linux"

# 3. Deploy on EC2
Write-Host "[3/4] Mengaktifkan binary di target container $Target..." -ForegroundColor Green
ssh -i "$EC2_KEY" "ubuntu@$EC2_HOST" "bash /home/ubuntu/deploy.sh $Target"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "  Deployment ke $Target SELESAI!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan
