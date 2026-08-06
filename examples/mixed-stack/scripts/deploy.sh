#!/usr/bin/env bash
# @ACP O DEPLOY.RELEASE
set -euo pipefail
source "$(dirname "$0")/env.sh"
deploy_release() {
  docker compose pull
  docker compose up -d
}
deploy_release
