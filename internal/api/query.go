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
	limit = clamp(limit, 1, 1000)
	if offset < 0 {
		offset = 0
	}

	var fromPtr, toPtr *time.Time
	if fromS != "" {
		t, err := time.Parse(time.RFC3339Nano, fromS)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from: expected RFC3339 timestamp"})
			return
		}
		fromPtr = &t
	}
	if toS != "" {
		t, err := time.Parse(time.RFC3339Nano, toS)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to: expected RFC3339 timestamp"})
			return
		}
		toPtr = &t
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

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
