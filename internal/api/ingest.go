package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/example/realtime-logs/internal/db"
	"github.com/example/realtime-logs/internal/models"
	"github.com/gin-gonic/gin"
)

type IngestHandler struct {
	Store db.Store
	// Broadcast is optional; if provided, emits new logs to subscribers.
	Broadcast func(entries []models.LogEntry)
	// Async pipeline
	queue chan []models.LogEntry
}

type ingestRequest struct {
	Items []models.LogEntry `json:"items"`
}

func (h *IngestHandler) Register(rg gin.IRoutes) {
	rg.POST("/ingest", h.handleIngest)
}

func (h *IngestHandler) handleIngest(c *gin.Context) {
	// Read raw body to support multiple JSON shapes
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}
	// Restore Body for downstream
	c.Request.Body = io.NopCloser(bytes.NewBuffer(data))

	var req ingestRequest
	if err := json.Unmarshal(data, &req); err != nil || len(req.Items) == 0 {
		// Try single object
		var single models.LogEntry
		if err2 := json.Unmarshal(data, &single); err2 == nil && single.OrgID != "" && single.Level != "" && single.Message != "" {
			req.Items = []models.LogEntry{single}
		}
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload: expected {items:[...]} or single object"})
		return
	}
	if err := validateEntries(req.Items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Enqueue for async processing
	if h.queue != nil {
		select {
		case h.queue <- req.Items:
			c.JSON(http.StatusAccepted, gin.H{"enqueued": len(req.Items)})
		default:
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "ingest queue is full"})
		}
		return
	}
	// Fallback to sync path
	if err := h.Store.Append(req.Items); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.Broadcast != nil {
		h.Broadcast(req.Items)
	}
	c.JSON(http.StatusAccepted, gin.H{"ingested": len(req.Items)})
}

func validateEntries(entries []models.LogEntry) error {
	for i, entry := range entries {
		if strings.TrimSpace(entry.OrgID) == "" {
			return &entryValidationError{index: i, field: "org_id"}
		}
		if strings.TrimSpace(entry.Level) == "" {
			return &entryValidationError{index: i, field: "level"}
		}
		if strings.TrimSpace(entry.Message) == "" {
			return &entryValidationError{index: i, field: "message"}
		}
	}
	return nil
}

type entryValidationError struct {
	index int
	field string
}

func (e *entryValidationError) Error() string {
	return "invalid payload: items[" + strconv.Itoa(e.index) + "]." + e.field + " is required"
}

// StartWorkers initializes background goroutines to store and broadcast logs.
func (h *IngestHandler) StartWorkers(n int) {
	h.StartWorkersWithQueue(n, 1024)
}

// StartWorkersWithQueue initializes workers using a custom queue capacity.
func (h *IngestHandler) StartWorkersWithQueue(n, queueSize int) {
	if n <= 0 {
		n = 2
	}
	if queueSize <= 0 {
		queueSize = 1024
	}
	if h.queue == nil {
		h.queue = make(chan []models.LogEntry, queueSize)
	}
	for i := 0; i < n; i++ {
		go func() {
			for batch := range h.queue {
				if len(batch) == 0 {
					continue
				}
				if err := h.Store.Append(batch); err != nil {
					log.Printf("ingest append failed: %v", err)
					continue
				}
				if h.Broadcast != nil {
					h.Broadcast(batch)
				}
			}
		}()
	}
}
