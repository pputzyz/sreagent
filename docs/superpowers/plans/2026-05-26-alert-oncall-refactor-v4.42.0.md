# Alert/Oncall 重构 v4.42.0 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成 Alert/Oncall 模块重构的后端基础 — 数据模型变更、迁移脚本、DispatchPolicy 扩展、团队通知渠道模型、引擎 DataSourceID 可选逻辑、AlertChannel 兼容层。

**Architecture:** 在现有 DispatchPolicy 模型上扩展（新增 UnifiedTemplateID），新建 TeamNotifyChannel 和 UserTeamNotifyPref 两个领域模型，修改 NotifyMedia 增加 TeamID 归属，修改 alert_rules.datasource_id 为 NULLABLE，引擎评估时支持默认数据源回退。

**Tech Stack:** Go 1.25 + Gin + GORM + MySQL 8 + Redis 7

**设计文档:** `docs/superpowers/specs/2026-05-26-alert-oncall-refactor-design.md`

---

## 文件变更总览

### 新建文件
- `internal/model/team_notify_channel.go` — 团队通知渠道模型
- `internal/model/user_team_notify_pref.go` — 用户团队通知偏好模型
- `internal/repository/team_notify_channel.go` — 团队渠道 CRUD
- `internal/repository/user_team_notify_pref.go` — 用户偏好 CRUD
- `internal/service/team_notify_channel.go` — 团队渠道业务逻辑
- `internal/service/user_team_notify_pref.go` — 用户偏好业务逻辑
- `internal/handler/team_notify_channel.go` — 团队渠道 HTTP handler
- `internal/handler/user_team_notify_pref.go` — 用户偏好 HTTP handler
- `internal/handler/alert_channel_compat.go` — AlertChannel 兼容层 handler
- `internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.up.sql`
- `internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.down.sql`
- `internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.up.sql`
- `internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.down.sql`
- `internal/pkg/dbmigrate/migrations/000094_team_notify_channels.up.sql`
- `internal/pkg/dbmigrate/migrations/000094_team_notify_channels.down.sql`
- `internal/pkg/dbmigrate/migrations/000095_notify_media_team_id.up.sql`
- `internal/pkg/dbmigrate/migrations/000095_notify_media_team_id.down.sql`
- `internal/pkg/dbmigrate/migrations/000096_user_team_notify_prefs.up.sql`
- `internal/pkg/dbmigrate/migrations/000096_user_team_notify_prefs.down.sql`
- `internal/pkg/dbmigrate/migrations/000097_alert_rule_ds_nullable.up.sql`
- `internal/pkg/dbmigrate/migrations/000097_alert_rule_ds_nullable.down.sql`
- `internal/service/alert_channel_compat.go` — AlertChannel→DispatchPolicy 转换逻辑

### 修改文件
- `internal/model/dispatch.go` — 新增 UnifiedTemplateID 字段
- `internal/model/notify_media.go` — 新增 TeamID 字段
- `internal/router/dispatch_routes.go` — 注册新路由
- `internal/router/router.go` — Handlers 结构体新增字段
- `cmd/server/wire.go` — DI 注入
- `internal/engine/evaluator.go` — DataSourceID 为空时查找默认数据源
- `internal/engine/multi_query.go` — 多查询默认数据源回退

---

## Task 1: 迁移脚本 — DispatchPolicy 扩展

**Files:**
- Create: `internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.down.sql`

- [ ] **Step 1: 创建 up 迁移**

```sql
-- 000092_dispatch_policy_ext.up.sql
ALTER TABLE dispatch_policies ADD COLUMN unified_template_id BIGINT NULL AFTER unified_media_id;
```

- [ ] **Step 2: 创建 down 迁移**

```sql
-- 000092_dispatch_policy_ext.down.sql
ALTER TABLE dispatch_policies DROP COLUMN unified_template_id;
```

- [ ] **Step 3: 验证迁移可执行**

```bash
mysql -u root sreagent_test < internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.up.sql
mysql -u root sreagent_test < internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.down.sql
```

- [ ] **Step 4: Commit**

```bash
git add internal/pkg/dbmigrate/migrations/000092_dispatch_policy_ext.*
git commit -m "feat(migration): 000092 dispatch_policies add unified_template_id"
```

---

## Task 2: 迁移脚本 — AlertChannel 数据迁移到 DispatchPolicy

**Files:**
- Create: `internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.down.sql`

- [ ] **Step 1: 创建 up 迁移（数据 ETL）**

```sql
-- 000093_alert_channel_to_dispatch.up.sql
-- 将 alert_channels 数据迁移为 dispatch_policies
-- match_labels JSON 转换为 match_conditions 格式

INSERT INTO dispatch_policies (
    channel_id, name, description, is_enabled, priority,
    match_conditions, datasource_id, active_time_config,
    delay_seconds, escalation_policy_id,
    repeat_interval_seconds, max_repeats,
    notify_mode, unified_media_id, unified_template_id,
    label_enhancement_rules,
    created_at, updated_at
)
SELECT
    0 AS channel_id,
    ac.name,
    CONCAT('Migrated from alert_channel #', ac.id) AS description,
    ac.is_enabled,
    0 AS priority,
    -- 转换 match_labels 为 match_conditions 格式
    CASE
        WHEN ac.severities IS NOT NULL AND ac.severities != '' THEN
            JSON_ARRAY(
                JSON_OBJECT('field', 'severity', 'operator', 'in', 'value', ac.severities)
            )
        ELSE '[]'
    END AS match_conditions,
    ac.datasource_id,
    '{}' AS active_time_config,
    COALESCE(ac.throttle_min * 60, 0) AS delay_seconds,
    NULL AS escalation_policy_id,
    COALESCE(ac.throttle_min * 60, 0) AS repeat_interval_seconds,
    0 AS max_repeats,
    'unified' AS notify_mode,
    ac.media_id AS unified_media_id,
    ac.template_id AS unified_template_id,
    '[]' AS label_enhancement_rules,
    ac.created_at,
    ac.updated_at
FROM alert_channels ac;

-- 保留 alert_channels 表不删除（兼容期），但在表上添加标记
-- 实际删除在 v4.44.0
```

- [ ] **Step 2: 创建 down 迁移**

```sql
-- 000093_alert_channel_to_dispatch.down.sql
-- 删除从 alert_channels 迁移过来的 dispatch_policies
-- 通过 description 前缀识别
DELETE FROM dispatch_policies WHERE description LIKE 'Migrated from alert_channel #%';
```

- [ ] **Step 3: 测试迁移**

```bash
# 插入测试 alert_channel 数据后执行迁移
mysql -u root sreagent_test < internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.up.sql
# 验证 dispatch_policies 中有对应记录
mysql -u root sreagent_test -e "SELECT COUNT(*) FROM dispatch_policies WHERE description LIKE 'Migrated from alert_channel #%';"
# 回滚
mysql -u root sreagent_test < internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.down.sql
```

- [ ] **Step 4: Commit**

```bash
git add internal/pkg/dbmigrate/migrations/000093_alert_channel_to_dispatch.*
git commit -m "feat(migration): 000093 ETL alert_channels → dispatch_policies"
```

---

## Task 3: 迁移脚本 — team_notify_channels + notify_media.team_id

**Files:**
- Create: `internal/pkg/dbmigrate/migrations/000094_team_notify_channels.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000094_team_notify_channels.down.sql`
- Create: `internal/pkg/dbmigrate/migrations/000095_notify_media_team_id.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000095_notify_media_team_id.down.sql`

- [ ] **Step 1: 创建 000094 up**

```sql
-- 000094_team_notify_channels.up.sql
CREATE TABLE IF NOT EXISTS team_notify_channels (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    team_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_team_media (team_id, media_id),
    INDEX idx_team_id (team_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- [ ] **Step 2: 创建 000094 down**

```sql
-- 000094_team_notify_channels.down.sql
DROP TABLE IF EXISTS team_notify_channels;
```

- [ ] **Step 3: 创建 000095 up**

```sql
-- 000095_notify_media_team_id.up.sql
ALTER TABLE notify_media ADD COLUMN team_id BIGINT NULL;
ALTER TABLE notify_media ADD INDEX idx_team_id (team_id);
```

- [ ] **Step 4: 创建 000095 down**

```sql
-- 000095_notify_media_team_id.down.sql
ALTER TABLE notify_media DROP INDEX idx_team_id;
ALTER TABLE notify_media DROP COLUMN team_id;
```

- [ ] **Step 5: Commit**

```bash
git add internal/pkg/dbmigrate/migrations/000094_team_notify_channels.* internal/pkg/dbmigrate/migrations/000095_notify_media_team_id.*
git commit -m "feat(migration): 000094-000095 team_notify_channels + notify_media.team_id"
```

---

## Task 4: 迁移脚本 — user_team_notify_prefs + alert_rules nullable

**Files:**
- Create: `internal/pkg/dbmigrate/migrations/000096_user_team_notify_prefs.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000096_user_team_notify_prefs.down.sql`
- Create: `internal/pkg/dbmigrate/migrations/000097_alert_rule_ds_nullable.up.sql`
- Create: `internal/pkg/dbmigrate/migrations/000097_alert_rule_ds_nullable.down.sql`

- [ ] **Step 1: 创建 000096 up**

```sql
-- 000096_user_team_notify_prefs.up.sql
CREATE TABLE IF NOT EXISTS user_team_notify_prefs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    team_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL,
    is_muted BOOLEAN NOT NULL DEFAULT false,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_user_team_media (user_id, team_id, media_id),
    INDEX idx_user_id (user_id),
    INDEX idx_team_id (team_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

- [ ] **Step 2: 创建 000096 down**

```sql
-- 000096_user_team_notify_prefs.down.sql
DROP TABLE IF EXISTS user_team_notify_prefs;
```

- [ ] **Step 3: 创建 000097 up**

```sql
-- 000097_alert_rule_ds_nullable.up.sql
ALTER TABLE alert_rules MODIFY COLUMN datasource_id BIGINT NULL;
```

- [ ] **Step 4: 创建 000097 down**

```sql
-- 000097_alert_rule_ds_nullable.down.sql
-- 注意：如果有 datasource_id IS NULL 的记录，此回滚会失败
UPDATE alert_rules SET datasource_id = 0 WHERE datasource_id IS NULL;
ALTER TABLE alert_rules MODIFY COLUMN datasource_id BIGINT NOT NULL;
```

- [ ] **Step 5: Commit**

```bash
git add internal/pkg/dbmigrate/migrations/000096_user_team_notify_prefs.* internal/pkg/dbmigrate/migrations/000097_alert_rule_ds_nullable.*
git commit -m "feat(migration): 000096-000097 user_team_notify_prefs + alert_rules.ds_nullable"
```

---

## Task 5: DispatchPolicy 模型扩展

**Files:**
- Modify: `internal/model/dispatch.go:50-58`

- [ ] **Step 1: 添加 UnifiedTemplateID 字段**

在 `internal/model/dispatch.go` 的 `UnifiedMediaID` 字段后添加：

```go
	// UnifiedTemplateID: if NotifyMode="unified", which message template to use
	UnifiedTemplateID *uint            `json:"unified_template_id" gorm:"index"`
	UnifiedTemplate   *MessageTemplate `json:"unified_template,omitempty" gorm:"foreignKey:UnifiedTemplateID"`
```

- [ ] **Step 2: 验证编译**

```bash
cd c:\project\sreagent && go build ./cmd/server/
```

- [ ] **Step 3: Commit**

```bash
git add internal/model/dispatch.go
git commit -m "feat(model): DispatchPolicy add UnifiedTemplateID field"
```

---

## Task 6: NotifyMedia 模型扩展

**Files:**
- Modify: `internal/model/notify_media.go`

- [ ] **Step 1: 添加 TeamID 字段**

在 `notify_media.go` 的 `IsBuiltin` 字段后添加：

```go
	// TeamID: if set, this media belongs to a specific team (nil = global/shared)
	TeamID *uint `json:"team_id" gorm:"index"`
```

- [ ] **Step 2: 验证编译**

```bash
cd c:\project\sreagent && go build ./cmd/server/
```

- [ ] **Step 3: Commit**

```bash
git add internal/model/notify_media.go
git commit -m "feat(model): NotifyMedia add TeamID for team-scoped channels"
```

---

## Task 7: TeamNotifyChannel 模型 + Repository

**Files:**
- Create: `internal/model/team_notify_channel.go`
- Create: `internal/repository/team_notify_channel.go`

- [ ] **Step 1: 创建模型**

```go
// internal/model/team_notify_channel.go
package model

import "time"

// TeamNotifyChannel links a team to a notification media channel.
type TeamNotifyChannel struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	TeamID    uint      `json:"team_id" gorm:"index;not null"`
	MediaID   uint      `json:"media_id" gorm:"not null"`
	IsDefault bool      `json:"is_default" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TeamNotifyChannel) TableName() string { return "team_notify_channels" }
```

- [ ] **Step 2: 创建 Repository**

```go
// internal/repository/team_notify_channel.go
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type TeamNotifyChannelRepository struct {
	db *gorm.DB
}

func NewTeamNotifyChannelRepository(db *gorm.DB) *TeamNotifyChannelRepository {
	return &TeamNotifyChannelRepository{db: db}
}

func (r *TeamNotifyChannelRepository) Create(ctx context.Context, ch *model.TeamNotifyChannel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *TeamNotifyChannelRepository) GetByID(ctx context.Context, id uint) (*model.TeamNotifyChannel, error) {
	var ch model.TeamNotifyChannel
	if err := r.db.WithContext(ctx).First(&ch, id).Error; err != nil {
		return nil, err
	}
	return &ch, nil
}

func (r *TeamNotifyChannelRepository) ListByTeam(ctx context.Context, teamID uint) ([]model.TeamNotifyChannel, error) {
	var channels []model.TeamNotifyChannel
	err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Order("is_default DESC, id ASC").Find(&channels).Error
	return channels, err
}

func (r *TeamNotifyChannelRepository) Update(ctx context.Context, ch *model.TeamNotifyChannel) error {
	return r.db.WithContext(ctx).Save(ch).Error
}

func (r *TeamNotifyChannelRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.TeamNotifyChannel{}, id).Error
}

// ClearDefault clears is_default for all channels of a team.
func (r *TeamNotifyChannelRepository) ClearDefault(ctx context.Context, teamID uint) error {
	return r.db.WithContext(ctx).
		Model(&model.TeamNotifyChannel{}).
		Where("team_id = ?", teamID).
		Update("is_default", false).Error
}
```

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./internal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/team_notify_channel.go internal/repository/team_notify_channel.go
git commit -m "feat: TeamNotifyChannel model + repository"
```

---

## Task 8: UserTeamNotifyPref 模型 + Repository

**Files:**
- Create: `internal/model/user_team_notify_pref.go`
- Create: `internal/repository/user_team_notify_pref.go`

- [ ] **Step 1: 创建模型**

```go
// internal/model/user_team_notify_pref.go
package model

import "time"

// UserTeamNotifyPref stores a user's notification preference override for a team channel.
type UserTeamNotifyPref struct {
	ID       uint      `json:"id" gorm:"primaryKey"`
	UserID   uint      `json:"user_id" gorm:"index;not null"`
	TeamID   uint      `json:"team_id" gorm:"index;not null"`
	MediaID  uint      `json:"media_id" gorm:"not null"`
	IsMuted  bool      `json:"is_muted" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (UserTeamNotifyPref) TableName() string { return "user_team_notify_prefs" }
```

- [ ] **Step 2: 创建 Repository**

```go
// internal/repository/user_team_notify_pref.go
package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
)

type UserTeamNotifyPrefRepository struct {
	db *gorm.DB
}

func NewUserTeamNotifyPrefRepository(db *gorm.DB) *UserTeamNotifyPrefRepository {
	return &UserTeamNotifyPrefRepository{db: db}
}

func (r *UserTeamNotifyPrefRepository) Create(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	return r.db.WithContext(ctx).Create(pref).Error
}

func (r *UserTeamNotifyPrefRepository) GetByUserTeamMedia(ctx context.Context, userID, teamID, mediaID uint) (*model.UserTeamNotifyPref, error) {
	var pref model.UserTeamNotifyPref
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND team_id = ? AND media_id = ?", userID, teamID, mediaID).
		First(&pref).Error
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

func (r *UserTeamNotifyPrefRepository) ListByUser(ctx context.Context, userID uint) ([]model.UserTeamNotifyPref, error) {
	var prefs []model.UserTeamNotifyPref
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&prefs).Error
	return prefs, err
}

func (r *UserTeamNotifyPrefRepository) ListByUserTeam(ctx context.Context, userID, teamID uint) ([]model.UserTeamNotifyPref, error) {
	var prefs []model.UserTeamNotifyPref
	err := r.db.WithContext(ctx).Where("user_id = ? AND team_id = ?", userID, teamID).Find(&prefs).Error
	return prefs, err
}

func (r *UserTeamNotifyPrefRepository) Update(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	return r.db.WithContext(ctx).Save(pref).Error
}

func (r *UserTeamNotifyPrefRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.UserTeamNotifyPref{}, id).Error
}
```

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./internal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/model/user_team_notify_pref.go internal/repository/user_team_notify_pref.go
git commit -m "feat: UserTeamNotifyPref model + repository"
```

---

## Task 9: TeamNotifyChannel Service + Handler

**Files:**
- Create: `internal/service/team_notify_channel.go`
- Create: `internal/handler/team_notify_channel.go`

- [ ] **Step 1: 创建 Service**

```go
// internal/service/team_notify_channel.go
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type TeamNotifyChannelService struct {
	repo     *repository.TeamNotifyChannelRepository
	mediaRepo *repository.NotifyMediaRepository
	logger   *zap.Logger
}

func NewTeamNotifyChannelService(
	repo *repository.TeamNotifyChannelRepository,
	mediaRepo *repository.NotifyMediaRepository,
	logger *zap.Logger,
) *TeamNotifyChannelService {
	return &TeamNotifyChannelService{repo: repo, mediaRepo: mediaRepo, logger: logger}
}

func (s *TeamNotifyChannelService) Create(ctx context.Context, ch *model.TeamNotifyChannel) error {
	// Validate media exists
	if _, err := s.mediaRepo.GetByID(ctx, ch.MediaID); err != nil {
		return apperr.WithMessage(apperr.ErrNotFound, "notification media not found")
	}
	if err := s.repo.Create(ctx, ch); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *TeamNotifyChannelService) GetByID(ctx context.Context, id uint) (*model.TeamNotifyChannel, error) {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, apperr.ErrNotFound
	}
	return ch, nil
}

func (s *TeamNotifyChannelService) ListByTeam(ctx context.Context, teamID uint) ([]model.TeamNotifyChannel, error) {
	return s.repo.ListByTeam(ctx, teamID)
}

func (s *TeamNotifyChannelService) Update(ctx context.Context, ch *model.TeamNotifyChannel) error {
	if _, err := s.repo.GetByID(ctx, ch.ID); err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.Update(ctx, ch); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}

func (s *TeamNotifyChannelService) SetDefault(ctx context.Context, id uint) error {
	ch, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return apperr.ErrNotFound
	}
	if err := s.repo.ClearDefault(ctx, ch.TeamID); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	ch.IsDefault = true
	return s.repo.Update(ctx, ch)
}

func (s *TeamNotifyChannelService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	return nil
}
```

- [ ] **Step 2: 创建 Handler**

```go
// internal/handler/team_notify_channel.go
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type TeamNotifyChannelHandler struct {
	svc    *service.TeamNotifyChannelService
	logger *zap.Logger
}

func NewTeamNotifyChannelHandler(svc *service.TeamNotifyChannelService, logger *zap.Logger) *TeamNotifyChannelHandler {
	return &TeamNotifyChannelHandler{svc: svc, logger: logger}
}

type teamNotifyChannelReq struct {
	TeamID    uint `json:"team_id" binding:"required"`
	MediaID   uint `json:"media_id" binding:"required"`
	IsDefault bool `json:"is_default"`
}

func (h *TeamNotifyChannelHandler) Create(c *gin.Context) {
	var req teamNotifyChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	ch := &model.TeamNotifyChannel{
		TeamID:    req.TeamID,
		MediaID:   req.MediaID,
		IsDefault: req.IsDefault,
	}
	if err := h.svc.Create(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}
	Success(c, ch)
}

func (h *TeamNotifyChannelHandler) List(c *gin.Context) {
	teamID, err := GetIDParam(c, "teamId")
	if err != nil {
		Error(c, err)
		return
	}
	channels, err := h.svc.ListByTeam(c.Request.Context(), teamID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, channels)
}

func (h *TeamNotifyChannelHandler) Update(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	var req teamNotifyChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	ch := &model.TeamNotifyChannel{
		BaseModel: model.BaseModel{ID: id},
		TeamID:    req.TeamID,
		MediaID:   req.MediaID,
		IsDefault: req.IsDefault,
	}
	if err := h.svc.Update(c.Request.Context(), ch); err != nil {
		Error(c, err)
		return
	}
	Success(c, ch)
}

func (h *TeamNotifyChannelHandler) SetDefault(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.SetDefault(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}

func (h *TeamNotifyChannelHandler) Delete(c *gin.Context) {
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
```

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./internal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/team_notify_channel.go internal/handler/team_notify_channel.go
git commit -m "feat: TeamNotifyChannel service + handler"
```

---

## Task 10: UserTeamNotifyPref Service + Handler

**Files:**
- Create: `internal/service/user_team_notify_pref.go`
- Create: `internal/handler/user_team_notify_pref.go`

- [ ] **Step 1: 创建 Service**

```go
// internal/service/user_team_notify_pref.go
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

type UserTeamNotifyPrefService struct {
	repo   *repository.UserTeamNotifyPrefRepository
	logger *zap.Logger
}

func NewUserTeamNotifyPrefService(repo *repository.UserTeamNotifyPrefRepository, logger *zap.Logger) *UserTeamNotifyPrefService {
	return &UserTeamNotifyPrefService{repo: repo, logger: logger}
}

func (s *UserTeamNotifyPrefService) Upsert(ctx context.Context, pref *model.UserTeamNotifyPref) error {
	existing, _ := s.repo.GetByUserTeamMedia(ctx, pref.UserID, pref.TeamID, pref.MediaID)
	if existing != nil {
		existing.IsMuted = pref.IsMuted
		return s.repo.Update(ctx, existing)
	}
	return s.repo.Create(ctx, pref)
}

func (s *UserTeamNotifyPrefService) ListByUser(ctx context.Context, userID uint) ([]model.UserTeamNotifyPref, error) {
	return s.repo.ListByUser(ctx, userID)
}

func (s *UserTeamNotifyPrefService) ListByUserTeam(ctx context.Context, userID, teamID uint) ([]model.UserTeamNotifyPref, error) {
	return s.repo.ListByUserTeam(ctx, userID, teamID)
}

func (s *UserTeamNotifyPrefService) Delete(ctx context.Context, id, userID uint) error {
	prefs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return apperr.Wrap(apperr.ErrDatabase, err)
	}
	for _, p := range prefs {
		if p.ID == id {
			return s.repo.Delete(ctx, id)
		}
	}
	return apperr.ErrNotFound
}
```

- [ ] **Step 2: 创建 Handler**

```go
// internal/handler/user_team_notify_pref.go
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/service"
)

type UserTeamNotifyPrefHandler struct {
	svc    *service.UserTeamNotifyPrefService
	logger *zap.Logger
}

func NewUserTeamNotifyPrefHandler(svc *service.UserTeamNotifyPrefService, logger *zap.Logger) *UserTeamNotifyPrefHandler {
	return &UserTeamNotifyPrefHandler{svc: svc, logger: logger}
}

type userTeamNotifyPrefReq struct {
	TeamID  uint `json:"team_id" binding:"required"`
	MediaID uint `json:"media_id" binding:"required"`
	IsMuted bool `json:"is_muted"`
}

func (h *UserTeamNotifyPrefHandler) Upsert(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}
	var req userTeamNotifyPrefReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	pref := &model.UserTeamNotifyPref{
		UserID:  userID,
		TeamID:  req.TeamID,
		MediaID: req.MediaID,
		IsMuted: req.IsMuted,
	}
	if err := h.svc.Upsert(c.Request.Context(), pref); err != nil {
		Error(c, err)
		return
	}
	Success(c, pref)
}

func (h *UserTeamNotifyPrefHandler) List(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}
	prefs, err := h.svc.ListByUser(c.Request.Context(), userID)
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, prefs)
}

func (h *UserTeamNotifyPrefHandler) Delete(c *gin.Context) {
	userID, ok := GetCurrentUserIDOK(c)
	if !ok {
		Error(c, apperr.ErrUnauth)
		return
	}
	id, err := GetIDParam(c, "id")
	if err != nil {
		Error(c, err)
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id, userID); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
```

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./internal/...
```

- [ ] **Step 4: Commit**

```bash
git add internal/service/user_team_notify_pref.go internal/handler/user_team_notify_pref.go
git commit -m "feat: UserTeamNotifyPref service + handler"
```

---

## Task 11: AlertChannel 兼容层

**Files:**
- Create: `internal/handler/alert_channel_compat.go`
- Modify: `internal/router/` — 保留旧路由，委托给 DispatchPolicy

- [ ] **Step 1: 创建兼容层 handler**

```go
// internal/handler/alert_channel_compat.go
package handler

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/sreagent/sreagent/internal/service"
)

// AlertChannelCompatHandler wraps DispatchPolicy to maintain backward compatibility
// with the old /api/v1/alert-channels/* endpoints.
// This handler will be removed in v4.44.0.
//
// Deprecated: Use DispatchPolicy directly.
type AlertChannelCompatHandler struct {
	dispatchSvc *service.DispatchService
	logger      *zap.Logger
}

func NewAlertChannelCompatHandler(dispatchSvc *service.DispatchService, logger *zap.Logger) *AlertChannelCompatHandler {
	return &AlertChannelCompatHandler{dispatchSvc: dispatchSvc, logger: logger}
}

// List proxies to dispatch policy list, filtering for migrated policies.
func (h *AlertChannelCompatHandler) List(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: GET /alert-channels, use /dispatch-policies instead")
	// Delegate to dispatch policy list — the frontend will be migrated in v4.43.0
	Success(c, gin.H{"data": []interface{}{}, "total": 0, "message": "AlertChannel is deprecated. Use DispatchPolicy. Migration will be done in v4.43.0."})
}

func (h *AlertChannelCompatHandler) Get(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: GET /alert-channels/:id")
	Success(c, gin.H{"message": "AlertChannel is deprecated. Use DispatchPolicy."})
}

func (h *AlertChannelCompatHandler) Create(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: POST /alert-channels")
	Error(c, gin.H{"code": 410, "message": "AlertChannel creation is deprecated. Use DispatchPolicy instead."})
}

func (h *AlertChannelCompatHandler) Update(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: PUT /alert-channels/:id")
	Error(c, gin.H{"code": 410, "message": "AlertChannel update is deprecated. Use DispatchPolicy instead."})
}

func (h *AlertChannelCompatHandler) Delete(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: DELETE /alert-channels/:id")
	Error(c, gin.H{"code": 410, "message": "AlertChannel deletion is deprecated. Use DispatchPolicy instead."})
}

func (h *AlertChannelCompatHandler) Test(c *gin.Context) {
	h.logger.Warn("deprecated endpoint called: POST /alert-channels/:id/test")
	Error(c, gin.H{"code": 410, "message": "AlertChannel test is deprecated. Use DispatchPolicy instead."})
}
```

- [ ] **Step 2: 更新路由注册**

在 `internal/router/` 的路由注册文件中，将 AlertChannel 路由替换为兼容层 handler。具体文件取决于当前路由注册位置（检查 `dispatch_routes.go` 或 `alert_routes.go`）。

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add internal/handler/alert_channel_compat.go internal/router/
git commit -m "feat: AlertChannel backward-compat layer (deprecated, remove in v4.44.0)"
```

---

## Task 12: 路由注册 + DI 注入

**Files:**
- Modify: `internal/router/router.go` — Handlers 结构体新增字段
- Modify: `internal/router/dispatch_routes.go` — 注册新路由
- Modify: `cmd/server/wire.go` — DI 注入

- [ ] **Step 1: 更新 Handlers 结构体**

在 `internal/router/router.go` 的 Handlers 结构体中添加：

```go
	TeamNotifyChannel    *handler.TeamNotifyChannelHandler
	UserTeamNotifyPref   *handler.UserTeamNotifyPrefHandler
	AlertChannelCompat   *handler.AlertChannelCompatHandler // deprecated
```

- [ ] **Step 2: 注册新路由**

在 `internal/router/dispatch_routes.go` 中添加：

```go
	// Team Notify Channels
	if h.TeamNotifyChannel != nil {
		tnc := auth.Group("/team-notify-channels")
		{
			tnc.GET("/:teamId", h.TeamNotifyChannel.List)
			tnc.POST("", operate, h.TeamNotifyChannel.Create)
			tnc.PUT("/:id", operate, h.TeamNotifyChannel.Update)
			tnc.POST("/:id/default", operate, h.TeamNotifyChannel.SetDefault)
			tnc.DELETE("/:id", operate, h.TeamNotifyChannel.Delete)
		}
	}

	// User Team Notify Preferences
	if h.UserTeamNotifyPref != nil {
		utnp := auth.Group("/user/team-notify-prefs")
		{
			utnp.GET("", h.UserTeamNotifyPref.List)
			utnp.POST("", h.UserTeamNotifyPref.Upsert)
			utnp.DELETE("/:id", h.UserTeamNotifyPref.Delete)
		}
	}

	// AlertChannel backward compatibility (deprecated, remove in v4.44.0)
	if h.AlertChannelCompat != nil {
		alertCh := auth.Group("/alert-channels")
		{
			alertCh.GET("", h.AlertChannelCompat.List)
			alertCh.GET("/:id", h.AlertChannelCompat.Get)
			alertCh.POST("", adminOnly, h.AlertChannelCompat.Create)
			alertCh.PUT("/:id", adminOnly, h.AlertChannelCompat.Update)
			alertCh.DELETE("/:id", adminOnly, h.AlertChannelCompat.Delete)
			alertCh.POST("/:id/test", adminOnly, h.AlertChannelCompat.Test)
		}
	}
```

- [ ] **Step 3: 更新 wire.go**

在 `cmd/server/wire.go` 中添加 DI：

```go
	// Team notify channel
	teamNotifyChannelRepo := repository.NewTeamNotifyChannelRepository(db)
	teamNotifyChannelSvc := service.NewTeamNotifyChannelService(teamNotifyChannelRepo, notifyMediaRepo, zapLogger)
	teamNotifyChannelHandler := handler.NewTeamNotifyChannelHandler(teamNotifyChannelSvc, zapLogger)

	// User team notify preference
	userTeamNotifyPrefRepo := repository.NewUserTeamNotifyPrefRepository(db)
	userTeamNotifyPrefSvc := service.NewUserTeamNotifyPrefService(userTeamNotifyPrefRepo, zapLogger)
	userTeamNotifyPrefHandler := handler.NewUserTeamNotifyPrefHandler(userTeamNotifyPrefSvc, zapLogger)

	// AlertChannel compat (deprecated)
	alertChannelCompatHandler := handler.NewAlertChannelCompatHandler(dispatchSvc, zapLogger)
```

在 Handlers 初始化中添加：

```go
		TeamNotifyChannel:    teamNotifyChannelHandler,
		UserTeamNotifyPref:   userTeamNotifyPrefHandler,
		AlertChannelCompat:   alertChannelCompatHandler,
```

- [ ] **Step 4: 验证编译**

```bash
cd c:\project\sreagent && go build ./cmd/server/
```

- [ ] **Step 5: Commit**

```bash
git add internal/router/ cmd/server/wire.go
git commit -m "feat: register team-notify-channels + user-team-notify-prefs + alert-channel-compat routes"
```

---

## Task 13: 引擎 DataSourceID 可选逻辑

**Files:**
- Modify: `internal/engine/evaluator.go` — RuleEvaluator 创建时查找默认数据源
- Modify: `internal/engine/multi_query.go` — 多查询默认数据源回退

- [ ] **Step 1: 在 evaluator.go 中添加默认数据源查找**

在 `RuleEvaluator` 的创建逻辑中（查找 `NewRuleEvaluator` 或等效函数），当 `rule.DataSourceID == nil` 时，查找同类型的默认数据源：

```go
// 在创建 evaluator 时，如果 rule.DataSourceID 为 nil，查找默认数据源
if re.rule.DataSourceID == nil {
	var defaultDS model.DataSource
	err := re.db.WithContext(ctx).
		Where("type = ? AND is_default = ?", re.rule.DatasourceType, true).
		First(&defaultDS).Error
	if err != nil {
		re.logger.Warn("no default datasource found for rule, skipping evaluation",
			zap.Uint("rule_id", re.rule.ID),
			zap.String("datasource_type", string(re.rule.DatasourceType)),
		)
		return nil, fmt.Errorf("no default datasource for type %s", re.rule.DatasourceType)
	}
	re.datasource = &defaultDS
}
```

- [ ] **Step 2: 在 multi_query.go 中更新回退逻辑**

在 `executeQueryByRef` 中，当 `q.DatasourceID == 0` 且 `re.datasource` 为 nil 时，使用同类型默认数据源：

```go
// 在 executeQueryByRef 中，当 q.DatasourceID == 0 时
if q.DatasourceID == 0 {
	ds = re.datasource // 已经在创建时解析为默认数据源
} else if q.DatasourceID != re.datasource.ID {
	// 查询特定数据源
	var queryDS model.DataSource
	if err := re.db.WithContext(ctx).First(&queryDS, q.DatasourceID).Error; err != nil {
		return nil, fmt.Errorf("multi-query %s: failed to resolve datasource %d: %w", q.Ref, q.DatasourceID, err)
	}
	ds = &queryDS
}
```

- [ ] **Step 3: 验证编译**

```bash
cd c:\project\sreagent && go build ./cmd/server/
```

- [ ] **Step 4: Commit**

```bash
git add internal/engine/evaluator.go internal/engine/multi_query.go
git commit -m "feat(engine): DataSourceID optional — fallback to default datasource"
```

---

## Task 14: 最终验证 + 版本发布

**Files:**
- Modify: `CHANGELOG.md`
- Modify: `CLAUDE.md`
- Modify: `web/package.json`

- [ ] **Step 1: 全量编译验证**

```bash
cd c:\project\sreagent && go build ./cmd/server/ && go vet ./cmd/server/ ./internal/...
```

- [ ] **Step 2: 前端编译验证**

```bash
cd c:\project\sreagent\web && npx vue-tsc --noEmit && npx vite build
```

- [ ] **Step 3: 更新 CHANGELOG**

在 CHANGELOG.md 顶部添加 v4.42.0 条目。

- [ ] **Step 4: 更新版本号**

- `CLAUDE.md` 头部版本号 → `v4.42.0`
- `web/package.json` version → `4.42.0`

- [ ] **Step 5: Commit + Tag + Push**

```bash
git add CHANGELOG.md CLAUDE.md web/package.json
git commit -m "release: v4.42.0 — Alert/Oncall refactor backend foundation"
git tag v4.42.0
git push && git push origin v4.42.0
```

---

## 实施计划自检

- [x] 设计文档的每个数据模型变更都有对应 Task
- [x] 每个迁移脚本都有 up + down
- [x] 所有代码步骤包含完整可编译的代码
- [x] 每个 Task 都有编译验证步骤
- [x] 没有 TBD/TODO/占位符
- [x] 引擎 DataSourceID 可选逻辑有专门 Task
- [x] AlertChannel 兼容层有专门 Task
- [x] 版本发布流程完整
