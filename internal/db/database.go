package db

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/example/realtime-logs/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// QueryParams defines filters for querying logs from the store
type QueryParams struct {
	OrgID  string
	Level  string
	Q      string // substring search in Message
	From   *time.Time
	To     *time.Time
	Limit  int
	Offset int
}

// Store is a generic interface to append and query logs.
type Store interface {
	Append(entries []models.LogEntry) error
	Query(params QueryParams) (items []models.LogEntry, total int)
}

// MemoryStore is a simple in-memory thread-safe store for logs.
type MemoryStore struct {
	mu       sync.RWMutex
	logs     []models.LogEntry
	capacity int
	nextID   uint
}

// NewMemoryStore creates a new MemoryStore with the given capacity.
func NewMemoryStore(capacity int) *MemoryStore {
	if capacity <= 0 {
		capacity = 10000
	}
	return &MemoryStore{
		logs:     make([]models.LogEntry, 0, capacity),
		capacity: capacity,
		nextID:   1,
	}
}

// Append adds logs to the store, trimming to capacity if necessary.
func (m *MemoryStore) Append(entries []models.LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	normalizeEntries(entries)
	for i := range entries {
		entries[i].ID = m.nextID
		m.nextID++
		m.logs = append(m.logs, entries[i])
	}
	// trim to capacity (drop oldest)
	if len(m.logs) > m.capacity {
		over := len(m.logs) - m.capacity
		m.logs = m.logs[over:]
	}
	return nil
}

// Query filters logs with simple in-memory scan and pagination.
func (m *MemoryStore) Query(p QueryParams) ([]models.LogEntry, int) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var filtered []models.LogEntry
	wantLevel := strings.TrimSpace(strings.ToLower(p.Level))
	containsQ := strings.TrimSpace(strings.ToLower(p.Q))
	for _, lg := range m.logs {
		if p.OrgID != "" && lg.OrgID != p.OrgID {
			continue
		}
		if wantLevel != "" && strings.ToLower(lg.Level) != wantLevel {
			continue
		}
		if containsQ != "" && !strings.Contains(strings.ToLower(lg.Message), containsQ) {
			continue
		}
		// time range filter (parse lazily)
		if p.From != nil && lg.Ts.Before(*p.From) {
			continue
		}
		if p.To != nil && lg.Ts.After(*p.To) {
			continue
		}
		filtered = append(filtered, lg)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].Ts.After(filtered[j].Ts)
	})
	total := len(filtered)
	// pagination
	start := p.Offset
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + p.Limit
	if p.Limit <= 0 || end > total {
		end = total
	}
	return filtered[start:end], total
}

// PostgresStore persists logs in PostgreSQL using GORM.
type PostgresStore struct {
	db *gorm.DB
}

// NewPostgresStore connects to PostgreSQL using the provided DSN and migrates schema.
func NewPostgresStore(dsn string) (*PostgresStore, error) {
	gdb, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	if err := gdb.WithContext(context.Background()).AutoMigrate(&models.LogEntry{}); err != nil {
		return nil, err
	}
	return &PostgresStore{db: gdb}, nil
}

func (p *PostgresStore) Append(entries []models.LogEntry) error {
	normalizeEntries(entries)
	return p.db.WithContext(context.Background()).Create(&entries).Error
}

func normalizeEntries(entries []models.LogEntry) {
	now := time.Now().UTC()
	for i := range entries {
		if strings.TrimSpace(entries[i].Timestamp) == "" {
			entries[i].Timestamp = now.Format(time.RFC3339Nano)
		}
		if t, err := time.Parse(time.RFC3339Nano, entries[i].Timestamp); err == nil {
			entries[i].Ts = t.UTC()
		} else {
			entries[i].Ts = now
			entries[i].Timestamp = now.Format(time.RFC3339Nano)
		}
	}
}

func (p *PostgresStore) Query(qp QueryParams) ([]models.LogEntry, int) {
	ctx := context.Background()
	q := p.db.WithContext(ctx).Model(&models.LogEntry{})
	if qp.OrgID != "" {
		q = q.Where("org_id = ?", qp.OrgID)
	}
	if qp.Level != "" {
		q = q.Where("LOWER(level) = LOWER(?)", qp.Level)
	}
	if qp.Q != "" {
		like := "%" + strings.ToLower(qp.Q) + "%"
		q = q.Where("LOWER(message) LIKE ?", like)
	}
	if qp.From != nil {
		q = q.Where("ts >= ?", qp.From)
	}
	if qp.To != nil {
		q = q.Where("ts <= ?", qp.To)
	}
	var total int64
	q.Count(&total)

	limit := qp.Limit
	if limit <= 0 {
		limit = 100
	}
	offset := qp.Offset
	if offset < 0 {
		offset = 0
	}

	var items []models.LogEntry
	q.Order("ts DESC").Limit(limit).Offset(offset).Find(&items)
	return items, int(total)
}
