package handler

import (
	"context"
	"fmt"
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
	dsIDs, err := parseDatasourceIDs(c.Query("datasource_id"))
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
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
	dsIDs, err := parseDatasourceIDs(c.Query("datasource_id"))
	if err != nil {
		Error(c, apperr.WithMessage(apperr.ErrInvalidParam, err.Error()))
		return
	}
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

// parseDatasourceIDs parses a comma-separated list of datasource IDs.
// Returns an error if any value is not a valid positive integer (B1-15).
func parseDatasourceIDs(raw string) ([]uint, error) {
	if raw == "" {
		return nil, nil
	}
	var ids []uint
	for _, s := range strings.Split(raw, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid datasource_id %q: must be a positive integer", s)
		}
		if v == 0 {
			return nil, fmt.Errorf("invalid datasource_id: must be > 0")
		}
		ids = append(ids, uint(v))
	}
	return ids, nil
}
