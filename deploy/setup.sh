#!/usr/bin/env bash
# ==============================================================================
# deploy/setup.sh — One-time server bootstrap script
#
# Run this ONCE on a fresh Ubuntu 22.04+ server as root (or with sudo):
#   bash setup.sh --domain your.domain.com
#
# What it does:
#   1. Updates the system and installs required packages
#   2. Installs PostgreSQL and creates the app database + user
#   3. Installs Node.js (for pnpm / build artifacts — not needed on server,
#      but kept as reference; frontend is built in CI and copied as static files)
#   4. Creates a dedicated non-root system user "bingo" for running the service
#   5. Creates directory structure
#   6. Installs nginx and deploys the nginx config
#   7. Obtains a Let's Encrypt SSL certificate via certbot
#   8. Installs the systemd service unit
#
# After running this script, push your first deploy via GitHub Actions.
# ==============================================================================
set -euo pipefail

# ── Argument parsing ──────────────────────────────────────────────────────────
DOMAIN=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --domain) DOMAIN="$2"; shift 2 ;;
        *) echo "Unknown argument: $1"; exit 1 ;;
    esac
done

if [[ -z "$DOMAIN" ]]; then
    echo "Usage: bash setup.sh --domain your.domain.com"
    exit 1
fi

echo "==> Setting up Bingo server for domain: $DOMAIN"

# ── 1. System update + base packages ─────────────────────────────────────────
echo "==> [1/7] Updating system packages..."
apt-get update -y
apt-get upgrade -y
apt-get install -y \
    curl wget git unzip \
    build-essential \
    ca-certificates gnupg lsb-release \
    ufw

# ── 2. PostgreSQL ─────────────────────────────────────────────────────────────
echo "==> [2/7] Installing PostgreSQL..."
apt-get install -y postgresql postgresql-contrib

# Start and enable PostgreSQL service
systemctl enable postgresql
systemctl start postgresql

# Create database and user.
# Credentials are set here; the same values must be put into GitHub Secrets
# as DB_PASSWORD and DB_NAME so that the Go app can connect.
DB_NAME="bingo"
DB_USER="bingo"
# Generate a random password or set it explicitly.
# For first-time setup, set DB_PASSWORD env var before running this script:
#   DB_PASSWORD=mysecret bash setup.sh --domain ...
DB_PASSWORD="${DB_PASSWORD:-$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 32)}"

echo "==> Creating PostgreSQL user '$DB_USER' and database '$DB_NAME'..."
sudo -u postgres psql -c "CREATE USER $DB_USER WITH PASSWORD '$DB_PASSWORD';" 2>/dev/null || \
    echo "    (user already exists, skipping)"
sudo -u postgres psql -c "CREATE DATABASE $DB_NAME OWNER $DB_USER;" 2>/dev/null || \
    echo "    (database already exists, skipping)"

echo ""
echo "    ┌─────────────────────────────────────────────────────────┐"
echo "    │  DB_CONNECTION_STRING for prod.env / GitHub Secret:     │"
echo "    │  postgres://$DB_USER:$DB_PASSWORD@localhost:5432/$DB_NAME │"
echo "    └─────────────────────────────────────────────────────────┘"
echo "    Save this connection string now — it won't be shown again."
echo ""

# ── 3. Create system user "bingo" ─────────────────────────────────────────────
echo "==> [3/7] Creating system user 'bingo'..."
id bingo &>/dev/null || useradd --system --shell /bin/bash --create-home bingo

# ── 4. Create directory structure ─────────────────────────────────────────────
echo "==> [4/7] Creating directory structure..."

# Config directory — the prod.env file will be placed here by GitHub Actions
# (or manually on first deploy)
mkdir -p /etc/bingo
chown bingo:bingo /etc/bingo
chmod 750 /etc/bingo

# Application binary directory
mkdir -p /opt/bingo
chown bingo:bingo /opt/bingo

# Frontend static files — nginx serves from here
mkdir -p /var/www/bingo
chown bingo:bingo /var/www/bingo

# Certbot webroot for ACME challenges
mkdir -p /var/www/certbot

# ── 5. nginx ──────────────────────────────────────────────────────────────────
echo "==> [5/7] Installing and configuring nginx..."
apt-get install -y nginx

# Copy the nginx config, substituting the real domain
# This script expects nginx.conf to be in the same directory as setup.sh
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
sed "s/YOUR_DOMAIN/$DOMAIN/g" "$SCRIPT_DIR/nginx.conf" \
    > /etc/nginx/sites-available/bingo

# Enable the site
ln -sf /etc/nginx/sites-available/bingo /etc/nginx/sites-enabled/bingo

# Remove the default nginx site if it exists
rm -f /etc/nginx/sites-enabled/default

# Validate config syntax
nginx -t

systemctl enable nginx
systemctl restart nginx

# ── 6. Certbot / Let's Encrypt SSL ────────────────────────────────────────────
echo "==> [6/7] Installing certbot and obtaining SSL certificate..."
apt-get install -y certbot python3-certbot-nginx

# Obtain certificate. --non-interactive requires email and domain.
# Set CERTBOT_EMAIL env var before running this script:
#   CERTBOT_EMAIL=you@example.com DB_PASSWORD=... bash setup.sh --domain ...
CERTBOT_EMAIL="${CERTBOT_EMAIL:-admin@$DOMAIN}"

certbot --nginx \
    --non-interactive \
    --agree-tos \
    --email "$CERTBOT_EMAIL" \
    -d "$DOMAIN" \
    --redirect || echo "    Certbot failed — you can re-run: certbot --nginx -d $DOMAIN"

# Auto-renew cron (certbot installs a systemd timer, but just in case)
systemctl enable certbot.timer 2>/dev/null || true

# ── 7. systemd service ────────────────────────────────────────────────────────
echo "==> [7/7] Installing bingo-api systemd service..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cp "$SCRIPT_DIR/bingo-api.service" /etc/systemd/system/bingo-api.service

systemctl daemon-reload
systemctl enable bingo-api

# ── Firewall ──────────────────────────────────────────────────────────────────
echo "==> Configuring UFW firewall..."
ufw allow OpenSSH
ufw allow "Nginx Full"  # ports 80 + 443
ufw --force enable

echo ""
echo "==> Setup complete!"
echo "    Next steps:"
echo "    1. Create /etc/bingo/prod.env with all required env vars (see README)"
echo "    2. Add GitHub Secrets to your repository (see README)"
echo "    3. Push to main — GitHub Actions will deploy automatically"
