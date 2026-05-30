package handler

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
	apperr "github.com/sreagent/sreagent/internal/pkg/errors"

	"github.com/sreagent/sreagent/internal/service"
)

type LabelRegistryHandler struct {
	svc         *service.LabelRegistryService
	ctx         context.Context // server-level context, cancelled on shutdown
	syncRunning atomic.Bool
}

func NewLabelRegistryHandler(svc *service.LabelRegistryService, ctx context.Context) *LabelRegistryHandler {
	return &LabelRegistryHandler{svc: svc, ctx: ctx}
}

// GetValues godoc
// GET /label-registry/values?key=biz_project&datasource_id=1,2
func (h *LabelRegistryHandler) GetValues(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, "key is required"))
		return
	}
	dsIDs := parseDatasourceIDs(c.Query("datasource_id"))
	values, err := h.svc.GetValues(c.Request.Context(), key, dsIDs)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, err.Error()))
		return
	}
	Success(c, values)
}

// GetKeys godoc
// GET /label-registry/keys?datasource_id=1,2
func (h *LabelRegistryHandler) GetKeys(c *gin.Context) {
	dsIDs := parseDatasourceIDs(c.Query("datasource_id"))
	keys, err := h.svc.GetKeys(c.Request.Context(), dsIDs)
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrDatabase, err.Error()))
		return
	}
	Success(c, keys)
}

// Sync triggers an immediate sync (admin only).
// POST /label-registry/sync
func (h *LabelRegistryHandler) Sync(c *gin.Context) {
	if !h.syncRunning.CompareAndSwap(false, true) {
		Success(c, gin.H{"message": "sync already in progress"})
		return
	}
	go func() {
		defer h.syncRunning.Store(false)
		// Use the server-level context so sync is cancelled on shutdown.
		h.svc.SyncAll(h.ctx)
	}()
	Success(c, gin.H{"message": "sync triggered"})
}

func parseDatasourceIDs(raw string) []uint {
	if raw == "" {
		return nil
	}
	var ids []uint
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if v, err := strconv.ParseUint(s, 10, 64); err == nil {
			ids = append(ids, uint(v))
		}
	}
	return ids
}
