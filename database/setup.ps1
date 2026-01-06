# PostgreSQL Setup Script for Candlecore
# Run this script to set up the database

Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Candlecore PostgreSQL Setup" -ForegroundColor Cyan
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""

# Check if PostgreSQL is installed
$pgVersion = Get-Command psql -ErrorAction SilentlyContinue
if (-not $pgVersion) {
    Write-Host "ERROR: psql not found in PATH" -ForegroundColor Red
    Write-Host "Please ensure PostgreSQL is installed and added to PATH" -ForegroundColor Yellow
    exit 1
}

Write-Host "✓ PostgreSQL found" -ForegroundColor Green
Write-Host ""

# Get password for candlecore user
$password = Read-Host "Enter password for 'candlecore' PostgreSQL user" -AsSecureString
$passwordPlain = [System.Net.NetworkCredential]::new("", $password).Password

Write-Host ""
Write-Host "Step 1: Creating database and user..." -ForegroundColor Yellow

# Create SQL commands
$createUserSQL = @"
-- Create user if not exists
DO `$`$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_user WHERE usename = 'candlecore') THEN
        CREATE USER candlecore WITH PASSWORD '$passwordPlain';
    END IF;
END
`$`$;

-- Create database if not exists
SELECT 'CREATE DATABASE candlecore OWNER candlecore'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'candlecore')\gexec

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE candlecore TO candlecore;
"@

# Save to temp file
$tempFile = [System.IO.Path]::GetTempFileName()
Set-Content -Path $tempFile -Value $createUserSQL

# Execute as postgres superuser
Write-Host "Connecting to PostgreSQL as 'postgres' user..." -ForegroundColor Cyan
Write-Host "(You may be prompted for the postgres user password)" -ForegroundColor Cyan
psql -U postgres -f $tempFile

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to create user and database" -ForegroundColor Red
    Remove-Item $tempFile
    exit 1
}

Remove-Item $tempFile
Write-Host "✓ User and database created" -ForegroundColor Green
Write-Host ""

# Run schema
Write-Host "Step 2: Creating database schema..." -ForegroundColor Yellow
$env:PGPASSWORD = $passwordPlain
psql -U candlecore -d candlecore -f "database\schema.sql"

if ($LASTEXITCODE -ne 0) {
    Write-Host "ERROR: Failed to create schema" -ForegroundColor Red
    exit 1
}

Write-Host "✓ Schema created successfully" -ForegroundColor Green
Write-Host ""

# Verify setup
Write-Host "Step 3: Verifying setup..." -ForegroundColor Yellow
$verifySQL = "SELECT COUNT(*) FROM accounts;"
$result = psql -U candlecore -d candlecore -t -c $verifySQL

if ($LASTEXITCODE -eq 0) {
    Write-Host "✓ Database setup verified" -ForegroundColor Green
    Write-Host ""
    Write-Host "Initial account created with balance: 10000.0" -ForegroundColor Green
} else {
    Write-Host "WARNING: Could not verify setup" -ForegroundColor Yellow
}

# Clear password from env
$env:PGPASSWORD = $null

Write-Host ""
Write-Host "==================================" -ForegroundColor Cyan
Write-Host "Setup Complete!" -ForegroundColor Green
Write-Host "==================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Next steps:" -ForegroundColor Yellow
Write-Host "1. Update config.yaml:" -ForegroundColor Cyan
Write-Host "   database:" -ForegroundColor White
Write-Host "     enabled: true" -ForegroundColor White
Write-Host "     password: $passwordPlain" -ForegroundColor White
Write-Host ""
Write-Host "2. Or set environment variable:" -ForegroundColor Cyan
Write-Host "   `$env:CANDLECORE_DB_PASSWORD=`"$passwordPlain`"" -ForegroundColor White
Write-Host ""
Write-Host "3. Run Candlecore:" -ForegroundColor Cyan
Write-Host "   .\candlecore.exe" -ForegroundColor White
Write-Host ""
