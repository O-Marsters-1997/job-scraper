#!/usr/bin/env bash
set -euo pipefail

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

mkdir -p /opt/job-scraper/frontend/dist

ufw default deny incoming
ufw default allow outgoing
ufw allow OpenSSH
ufw --force enable

docker --version
caddy version
ufw status
