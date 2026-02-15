package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	apiKeyPrefix    = "mc_"
	apiKeyRandBytes = 32
	apiKeyDBPrefix  = 8 // Erste 8 Zeichen nach "mc_" als DB-Lookup-Prefix
)

// generateAPIKey erzeugt einen neuen API-Key mit Prefix, Hash und Lookup-Prefix.
func generateAPIKey() (raw, hash, prefix string, err error) {
	b := make([]byte, apiKeyRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	raw = apiKeyPrefix + hex.EncodeToString(b)
	hash = hashAPIKey(raw)
	prefix = raw[len(apiKeyPrefix) : len(apiKeyPrefix)+apiKeyDBPrefix]
	return raw, hash, prefix, nil
}

// hashAPIKey berechnet den SHA-256-Hash eines API-Keys.
func hashAPIKey(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// extractPrefix extrahiert den Lookup-Prefix aus einem rohen API-Key.
func extractPrefix(raw string) (string, error) {
	if len(raw) < len(apiKeyPrefix)+apiKeyDBPrefix {
		return "", fmt.Errorf("api key too short")
	}
	if raw[:len(apiKeyPrefix)] != apiKeyPrefix {
		return "", fmt.Errorf("invalid api key prefix")
	}
	return raw[len(apiKeyPrefix) : len(apiKeyPrefix)+apiKeyDBPrefix], nil
}
