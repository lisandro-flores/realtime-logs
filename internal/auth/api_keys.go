package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// SimpleKeys keeps a set of allowed API keys.
type SimpleKeys struct {
	allowed map[string]struct{}
}

// NewSimpleKeys initializes a SimpleKeys from a comma separated string.
func NewSimpleKeys(csv string) *SimpleKeys {
	s := &SimpleKeys{allowed: map[string]struct{}{}}
	for _, part := range strings.Split(csv, ",") {
		k := strings.TrimSpace(part)
		if k != "" {
			s.allowed[k] = struct{}{}
		}
	}
	return s
}

// Middleware returns a Gin middleware that validates X-API-Key header.
func (s *SimpleKeys) Middleware() gin.HandlerFunc {
	// If no keys configured, allow all (dev mode)
	if len(s.allowed) == 0 {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.GetHeader("X-API-Key"))
		if key == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing X-API-Key"})
			return
		}
		if _, ok := s.allowed[key]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid API key"})
			return
		}
		c.Next()
	}
}
