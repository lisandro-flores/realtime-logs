package models

import "time"

// LogEntry representa una entrada de log en la base de datos
type LogEntry struct {
	ID        uint      `gorm:"primaryKey"`
	OrgID     string    `json:"org_id" binding:"required"`
	Level     string    `json:"level" binding:"required"`
	Message   string    `json:"message" binding:"required"`
	Timestamp string    `json:"timestamp"`
	Ts        time.Time `json:"-" gorm:"index"`
}
