package auth

import "context"

type contextKey int

const (
	orgIDKey  contextKey = iota
	userIDKey
)

// WithTenant reichert den Context mit Tenant-Informationen an.
func WithTenant(ctx context.Context, orgID, userID string) context.Context {
	ctx = context.WithValue(ctx, orgIDKey, orgID)
	ctx = context.WithValue(ctx, userIDKey, userID)
	return ctx
}

// OrgIDFromContext gibt die Org-ID aus dem Context zurück.
// Gibt "", false zurück wenn kein Tenant gesetzt ist (z.B. Reconciler).
func OrgIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(orgIDKey).(string)
	return v, ok
}

// UserIDFromContext gibt die User-ID aus dem Context zurück.
// Gibt "", false zurück wenn kein User gesetzt ist.
func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(userIDKey).(string)
	return v, ok
}
