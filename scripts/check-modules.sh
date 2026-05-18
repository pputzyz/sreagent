#!/usr/bin/env bash
# check-modules.sh — verify MODULES.md claimed counts against actual codebase
# Usage: bash scripts/check-modules.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$REPO_ROOT"

# ── Parse claimed counts from MODULES.md ──────────────────────────────────────
HEADER_LINE=$(grep -E '^> 共 [0-9]+ 个 model' MODULES.md | head -1)
if [ -z "$HEADER_LINE" ]; then
  echo "ERROR: Cannot find claimed counts header in MODULES.md"
  echo '  Expected line like: > 共 36 个 model, 43 个 handler, 41 个 service, 33 个 repository'
  exit 1
fi

CLAIMED_MODELS=$(echo "$HEADER_LINE" | grep -oP '\d+(?= 个 model)')
CLAIMED_HANDLERS=$(echo "$HEADER_LINE" | grep -oP '\d+(?= 个 handler)')
CLAIMED_SERVICES=$(echo "$HEADER_LINE" | grep -oP '\d+(?= 个 service)')
CLAIMED_REPOS=$(echo "$HEADER_LINE" | grep -oP '\d+(?= 个 repository)')

echo "=== MODULES.md Sync Check ==="
echo "Claimed: model=$CLAIMED_MODELS, handler=$CLAIMED_HANDLERS, service=$CLAIMED_SERVICES, repository=$CLAIMED_REPOS"
echo ""

PASS=0
FAIL=0

check() {
  local name="$1" claimed="$2" actual="$3"
  if [ "$claimed" -eq "$actual" ]; then
    echo "  PASS  $name: claimed=$claimed actual=$actual"
    PASS=$((PASS + 1))
  else
    echo "  FAIL  $name: claimed=$claimed actual=$actual"
    FAIL=$((FAIL + 1))
  fi
}

# ── Count model types (unique struct names with TableName() receiver) ─────────
ACTUAL_MODELS=$(grep -rn 'func ([A-Za-z]*) TableName' internal/model/ --include='*.go' \
  | grep -v '_test.go' \
  | sed 's/.*func (\([A-Za-z]*\)).*/\1/' \
  | sort -u \
  | wc -l | tr -d ' ')
check "model types (TableName)" "$CLAIMED_MODELS" "$ACTUAL_MODELS"

# ── Count handler structs (*Handler) ──────────────────────────────────────────
ACTUAL_HANDLERS=$(grep -rn 'type [A-Za-z]*Handler struct' internal/handler/ --include='*.go' \
  | grep -v '_test.go' \
  | sed 's/.*type \([A-Za-z]*Handler\) struct.*/\1/' \
  | sort -u \
  | wc -l | tr -d ' ')
check "handler structs (*Handler)" "$CLAIMED_HANDLERS" "$ACTUAL_HANDLERS"

# ── Count service structs (*Service) ──────────────────────────────────────────
ACTUAL_SERVICES=$(grep -rn 'type [A-Za-z]*Service struct' internal/service/ --include='*.go' \
  | grep -v '_test.go' \
  | sed 's/.*type \([A-Za-z]*Service\) struct.*/\1/' \
  | sort -u \
  | wc -l | tr -d ' ')
check "service structs (*Service)" "$CLAIMED_SERVICES" "$ACTUAL_SERVICES"

# ── Count repository structs (*Repository) ────────────────────────────────────
ACTUAL_REPOS=$(grep -rn 'type [A-Za-z]*Repository struct' internal/repository/ --include='*.go' \
  | grep -v '_test.go' \
  | sed 's/.*type \([A-Za-z]*Repository\) struct.*/\1/' \
  | sort -u \
  | wc -l | tr -d ' ')
check "repository structs (*Repository)" "$CLAIMED_REPOS" "$ACTUAL_REPOS"

# ── Count migration .up.sql files ────────────────────────────────────────────
ACTUAL_MIGRATIONS=$(find internal/pkg/dbmigrate/migrations -name '*.up.sql' \
  | wc -l | tr -d ' ')
echo ""
echo "  INFO  migration .up.sql files: $ACTUAL_MIGRATIONS"

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "=== Result: $PASS passed, $FAIL failed ==="

if [ "$FAIL" -gt 0 ]; then
  echo "MODULES.md is out of sync with the actual codebase."
  echo "Update the counts in MODULES.md header: > 共 N 个 model, N 个 handler, N 个 service, N 个 repository"
  exit 1
fi

echo "MODULES.md is in sync."
exit 0
