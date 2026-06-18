package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/example/realtime-logs/internal/api"
	"github.com/example/realtime-logs/internal/auth"
	"github.com/example/realtime-logs/internal/db"
	"github.com/example/realtime-logs/internal/stream"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env if present
	_ = godotenv.Load()

	r := gin.New()
	if envBool("ACCESS_LOGS", true) {
		r.Use(gin.Logger())
	}
	r.Use(gin.Recovery())
	// Silence proxy warning for local/dev
	_ = r.SetTrustedProxies(nil)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Setup store (PostgreSQL via POSTGRES_DSN or fallback to in-memory) and streaming hub
	var store db.Store
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		if pg, err := db.NewPostgresStore(dsn); err == nil {
			log.Println("Using PostgreSQL store")
			store = pg
		} else {
			log.Printf("Postgres init failed, using memory store: %v", err)
			store = db.NewMemoryStore(10000)
		}
	} else {
		store = db.NewMemoryStore(10000)
	}
	hub := stream.NewHub()

	// Auth (optional via env API_KEYS="k1,k2")
	keys := auth.NewSimpleKeys(os.Getenv("API_KEYS"))
	apiGroup := r.Group("/api").Use(keys.Middleware())

	// API handlers
	ingest := &api.IngestHandler{Store: store, Broadcast: hub.Broadcast}
	workers := envInt("INGEST_WORKERS", 2)
	queueSize := envInt("INGEST_QUEUE_SIZE", 1024)
	ingest.StartWorkersWithQueue(workers, queueSize)
	log.Printf("Ingest workers=%d queue_size=%d", workers, queueSize)
	ingest.Register(apiGroup)
	// also expose root-level but with same auth
	secure := r.Group("/").Use(keys.Middleware())
	ingest.Register(secure)

	query := &api.QueryHandler{Store: store}
	query.Register(apiGroup)
	// also expose root-level but with same auth
	query.Register(secure)

	// WebSocket endpoint (protected by the same middleware; pass-through if no keys configured)
	r.GET("/ws", keys.Middleware(), hub.HandleWS)

	// Root info
	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, strings.TrimSpace(`Realtime Logs API
Endpoints:
GET  /health
GET  /            (this help)
WS   /ws          (stream logs as JSON array per message)
POST /api/ingest  (body: {"items": [{org_id, level, message, timestamp}]})
GET  /api/query   (params: org_id, level, q, from, to, limit, offset)
`))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server running on port %s", port)
	r.Run(":" + port)
}

func envInt(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func envBool(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}
