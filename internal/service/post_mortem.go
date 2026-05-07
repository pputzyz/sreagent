package service

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/sreagent/sreagent/internal/model"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"
	"github.com/sreagent/sreagent/internal/repository"
)

// PostMortemService manages incident post-mortems (故障复盘).
type PostMortemService struct {
	repo        *repository.PostMortemRepository
	incidentRepo *repository.IncidentRepository
	logger      *zap.Logger
}

func NewPostMortemService(
	repo *repository.PostMortemRepository,
	incidentRepo *repository.IncidentRepository,
	logger *zap.Logger,
) *PostMortemService {
	return &PostMortemService{repo: repo, incidentRepo: incidentRepo, logger: logger}
}

// GetOrCreate returns the post-mortem for an incident, creating a blank one if none exists.
func (s *PostMortemService) GetOrCreate(ctx context.Context, incidentID, userID uint) (*model.PostMortem, error) {
	pm, err := s.repo.GetByIncidentID(ctx, incidentID)
	if err == nil {
		return pm, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Verify incident exists
	inc, err := s.incidentRepo.GetByID(ctx, incidentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrIncidentNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	// Create blank draft
	pm = &model.PostMortem{
		IncidentID: incidentID,
		Title:      "故障复盘：" + inc.Title,
		Content:    defaultPostMortemTemplate(inc),
		Status:     "draft",
		AuthorID:   &userID,
	}
	if err := s.repo.Create(ctx, pm); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	s.logger.Info("post_mortem created", zap.Uint("incident_id", incidentID))
	return pm, nil
}

// Update saves content/title/status changes.
func (s *PostMortemService) Update(ctx context.Context, incidentID uint, title, content, status string) (*model.PostMortem, error) {
	pm, err := s.repo.GetByIncidentID(ctx, incidentID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.ErrNotFound
		}
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}

	if title != "" {
		pm.Title = title
	}
	if content != "" {
		pm.Content = content
	}
	if status != "" {
		pm.Status = status
		if status == "published" && pm.PublishedAt == nil {
			now := time.Now()
			pm.PublishedAt = &now
		}
	}

	if err := s.repo.Update(ctx, pm); err != nil {
		return nil, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return pm, nil
}

// List returns post-mortems filtered by channel and status.
func (s *PostMortemService) List(ctx context.Context, channelID uint, status string, page, pageSize int) ([]model.PostMortem, int64, error) {
	list, total, err := s.repo.List(ctx, channelID, status, page, pageSize)
	if err != nil {
		return nil, 0, apperr.Wrap(apperr.ErrDatabase, err)
	}
	return list, total, nil
}

// Publish marks a post-mortem as published.
func (s *PostMortemService) Publish(ctx context.Context, incidentID uint) (*model.PostMortem, error) {
	return s.Update(ctx, incidentID, "", "", "published")
}

// defaultPostMortemTemplate returns a Markdown template pre-filled with incident data.
func defaultPostMortemTemplate(inc *model.Incident) string {
	return `## 故障概述

**故障标题：** ` + inc.Title + `

**故障时间：** ` + inc.TriggeredAt.Format("2006-01-02 15:04:05") + `

**严重程度：** ` + string(inc.Severity) + `

---

## 故障影响

（描述影响的用户、服务、数据范围及持续时长）

---

## 根因分析

（描述故障的根本原因）

---

## 时间线

| 时间 | 事件 |
|------|------|
| ` + inc.TriggeredAt.Format("15:04") + ` | 故障触发 |

---

## 解决方案

（描述如何解决故障）

---

## 预防措施

（描述后续如何避免同类故障）

---

## 经验教训

（总结本次故障的经验教训）
`
}
