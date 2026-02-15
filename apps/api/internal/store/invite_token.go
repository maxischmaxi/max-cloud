package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const (
	inviteTokenPrefix    = "mci_"
	inviteTokenRandBytes = 32
	inviteTokenDBPrefix  = 8 // Erste 8 Zeichen nach "mci_" als DB-Lookup-Prefix
)

// generateInviteToken erzeugt einen neuen Invite-Token mit Prefix, Hash und Lookup-Prefix.
func generateInviteToken() (raw, hash, prefix string, err error) {
	b := make([]byte, inviteTokenRandBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", "", fmt.Errorf("generating random bytes: %w", err)
	}

	raw = inviteTokenPrefix + hex.EncodeToString(b)
	hash = hashInviteToken(raw)
	prefix = raw[len(inviteTokenPrefix) : len(inviteTokenPrefix)+inviteTokenDBPrefix]
	return raw, hash, prefix, nil
}

// hashInviteToken berechnet den SHA-256-Hash eines Invite-Tokens.
func hashInviteToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// extractInvitePrefix extrahiert den Lookup-Prefix aus einem rohen Invite-Token.
func extractInvitePrefix(raw string) (string, error) {
	if len(raw) < len(inviteTokenPrefix)+inviteTokenDBPrefix {
		return "", fmt.Errorf("invite token too short")
	}
	if raw[:len(inviteTokenPrefix)] != inviteTokenPrefix {
		return "", fmt.Errorf("invalid invite token prefix")
	}
	return raw[len(inviteTokenPrefix) : len(inviteTokenPrefix)+inviteTokenDBPrefix], nil
}
