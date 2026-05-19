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

// GetKeysByDatasource godoc
// GET /label-registry/datasource-keys?datasource_id=1
func (h *LabelRegistryHandler) GetKeysByDatasource(c *gin.Context) {
	dsIDStr := c.Query("datasource_id")
	if dsIDStr == "" {
		ErrorWithMessage(c, 10001, "datasource_id is required")
		return
	}
	dsID, err := strconv.ParseUint(dsIDStr, 10, 64)
	if err != nil {
		ErrorWithMessage(c, 10001, "invalid datasource_id")
		return
	}
	keys, err := h.svc.GetKeysByDatasource(c.Request.Context(), uint(dsID))
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, keys)
}

// GetValuesByDatasource godoc
// GET /label-registry/datasource-values?datasource_id=1&key=service
func (h *LabelRegistryHandler) GetValuesByDatasource(c *gin.Context) {
	dsIDStr := c.Query("datasource_id")
	if dsIDStr == "" {
		ErrorWithMessage(c, 10001, "datasource_id is required")
		return
	}
	dsID, err := strconv.ParseUint(dsIDStr, 10, 64)
	if err != nil {
		ErrorWithMessage(c, 10001, "invalid datasource_id")
		return
	}
	key := c.Query("key")
	if key == "" {
		ErrorWithMessage(c, 10001, "key is required")
		return
	}
	values, err := h.svc.GetValuesByDatasource(c.Request.Context(), uint(dsID), key)
	if err != nil {
		ErrorWithMessage(c, 50001, err.Error())
		return
	}
	Success(c, values)
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
