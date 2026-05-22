#!/bin/bash
set -e

# ============================================================
# SREAgent Docker Entrypoint
# K8s 部署时 initContainers 已保证 MySQL/Redis 可达，
# 这里只负责建库（首次部署）然后启动服务。
# 数据库表结构迁移由应用内嵌的 golang-migrate 自动完成。
# ============================================================

DB_HOST="${SREAGENT_DATABASE_HOST:-127.0.0.1}"
DB_PORT="${SREAGENT_DATABASE_PORT:-3306}"
DB_USER="${SREAGENT_DATABASE_USERNAME:-sreagent}"
DB_PASS="${SREAGENT_DATABASE_PASSWORD:-sreagent}"
DB_NAME="${SREAGENT_DATABASE_DATABASE:-sreagent}"

echo "============================================"
echo "  SREAgent - Intelligent SRE Platform"
echo "============================================"

# --- 确保数据库存在（首次部署时建库）---
CREATE_SQL="CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 使用 MYSQL_PWD 环境变量传递密码，避免命令行参数暴露在 /proc/cmdline
if MYSQL_PWD="${DB_PASS}" mysql -h"${DB_HOST}" -P"${DB_PORT}" -u"${DB_USER}" -e "${CREATE_SQL}" 2>/dev/null; then
  echo "[entrypoint] Database '${DB_NAME}' is ready."
else
  ROOT_PASS="${MYSQL_ROOT_PASSWORD:-}"
  if [ -n "${ROOT_PASS}" ]; then
    MYSQL_PWD="${ROOT_PASS}" mysql -h"${DB_HOST}" -P"${DB_PORT}" -uroot \
      -e "${CREATE_SQL}" 2>/dev/null \
      && echo "[entrypoint] Database '${DB_NAME}' created via root." \
      || echo "[entrypoint] WARNING: Could not create database, assuming it already exists."
  else
    echo "[entrypoint] WARNING: Could not create database (no root password). Assuming it already exists."
  fi
fi

# --- 启动服务（内嵌 golang-migrate 自动建表/升级）---
echo "[entrypoint] Starting SREAgent (:8080)..."
exec ./sreagent --config configs/config.yaml "$@"
