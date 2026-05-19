package handler

import (
	"context"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"

	"github.com/sreagent/sreagent/internal/service"
)

type LabelRegistryHandler struct {
	svc         *service.LabelRegistryService
	syncRunning atomic.Bool
}

func NewLabelRegistryHandler(svc *service.LabelRegistryService) *LabelRegistryHandler {
	return &LabelRegistryHandler{svc: svc}
}

// GetValues godoc
// GET /label-registry/values?key=biz_project&datasource_id=1,2
func (h *LabelRegistryHandler) GetValues(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		ErrorWithMessage(c, 10001, "key is required")
		return
	}
	dsIDs := parseDatasourceIDs(c.Query("datasource_id"))
	values, err := h.svc.GetValues(key, dsIDs)
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, values)
}

// GetKeys godoc
// GET /label-registry/keys?datasource_id=1,2
func (h *LabelRegistryHandler) GetKeys(c *gin.Context) {
	dsIDs := parseDatasourceIDs(c.Query("datasource_id"))
	keys, err := h.svc.GetKeys(dsIDs)
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
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
		// Use a detached context since the HTTP request returns immediately.
		h.svc.SyncAll(context.Background())
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
