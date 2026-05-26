package handler

import (
	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

// SiteInfoHandler manages site branding and customization settings.
type SiteInfoHandler struct {
	svc *service.SystemSettingService
}

// NewSiteInfoHandler creates a new SiteInfoHandler.
func NewSiteInfoHandler(svc *service.SystemSettingService) *SiteInfoHandler {
	return &SiteInfoHandler{svc: svc}
}

// Get returns the current site branding configuration.
func (h *SiteInfoHandler) Get(c *gin.Context) {
	cfg, err := h.svc.GetSiteInfo(c.Request.Context())
	if err != nil {
		Error(c, err)
		return
	}
	Success(c, cfg)
}

// Save persists site branding configuration.
func (h *SiteInfoHandler) Save(c *gin.Context) {
	var req service.SiteInfo
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
	if err := h.svc.SaveSiteInfo(c.Request.Context(), req); err != nil {
		Error(c, err)
		return
	}
	Success(c, nil)
}
