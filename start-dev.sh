#!/bin/bash
export SREAGENT_ADMIN_PASSWORD=admin123
export SREAGENT_DATABASE_PASSWORD=sreagent
export SREAGENT_DEV_SKIP_SSRF_CHECK=true
cd /c/project/sreagent
exec ./sreagent-server -config configs/config.yaml
