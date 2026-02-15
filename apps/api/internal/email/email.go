package email

import "context"

// Sender definiert die Schnittstelle für den E-Mail-Versand.
type Sender interface {
	SendInvite(ctx context.Context, toEmail, orgName, inviteToken string) error
}
