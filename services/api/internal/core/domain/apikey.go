package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

const (
	apiKeyMarker     = "dsg_"
	apiKeyBytes      = 32
	apiKeyPrefixSize = 8
)

type APIKey struct {
	ID        string
	Prefix    string
	CreatedAt time.Time
}

func NewAPIKeySecret() (string, error) {
	b := make([]byte, apiKeyBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return apiKeyMarker + base64.RawURLEncoding.EncodeToString(b), nil
}

func HashAPIKey(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func APIKeyPrefix(secret string) string {
	rest, ok := strings.CutPrefix(secret, apiKeyMarker)
	if !ok || len(rest) < apiKeyPrefixSize {
		return ""
	}
	return rest[:apiKeyPrefixSize]
}
