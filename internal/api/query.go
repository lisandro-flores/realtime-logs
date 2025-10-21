package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/example/realtime-logs/internal/db"
	"github.com/gin-gonic/gin"
)

type QueryHandler struct {
	Store db.Store
}

func (h *QueryHandler) Register(rg gin.IRoutes) {
	rg.GET("/query", h.handleQuery)
}

func (h *QueryHandler) handleQuery(c *gin.Context) {
	var (
		orgID  = c.Query("org_id")
		level  = c.Query("level")
		q      = c.Query("q")
		fromS  = c.Query("from")
		toS    = c.Query("to")
		limit  = parseIntDefault(c.Query("limit"), 100)
		offset = parseIntDefault(c.Query("offset"), 0)
	)

	var fromPtr, toPtr *time.Time
	if fromS != "" {
		if t, err := time.Parse(time.RFC3339Nano, fromS); err == nil {
			fromPtr = &t
		}
	}
	if toS != "" {
		if t, err := time.Parse(time.RFC3339Nano, toS); err == nil {
			toPtr = &t
		}
	}

	items, total := h.Store.Query(db.QueryParams{
		OrgID:  orgID,
		Level:  level,
		Q:      q,
		From:   fromPtr,
		To:     toPtr,
		Limit:  limit,
		Offset: offset,
	})
	c.JSON(http.StatusOK, gin.H{
		"total": total,
		"items": items,
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if v, err := strconv.Atoi(s); err == nil {
		return v
	}
	return def
}
