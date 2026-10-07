#!/usr/bin/env bash
set -euo pipefail

deploy_user="${SUDO_USER:?run with sudo as the user who will deploy}"

export DEBIAN_FRONTEND=noninteractive
apt-get update -q
apt-get install -yq ca-certificates curl gnupg rsync ufw debian-keyring debian-archive-keyring apt-transport-https

if ! command -v docker >/dev/null; then
  curl -fsSL https://get.docker.com | sh
fi

if ! command -v caddy >/dev/null; then
  curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key |
    gpg --dearmor --yes -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
  curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt >/etc/apt/sources.list.d/caddy-stable.list
  apt-get update -q
  apt-get install -yq caddy
fi

if ! command -v just >/dev/null; then
  curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to /usr/local/bin
fi

if ! command -v bun >/dev/null; then
  curl -fsSL https://bun.sh/install | BUN_INSTALL=/usr/local bash
fi

usermod -aG docker "$deploy_user"
mkdir -p /opt/job-scraper/frontend/dist
chown -R "$deploy_user": /opt/job-scraper

ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw --force enable

docker --version
caddy version
just --version
bun --version
ufw status
