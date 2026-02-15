package email

import (
	"context"
	"fmt"

	"github.com/resend/resend-go/v2"
)

// ResendSender versendet E-Mails über die Resend-API.
type ResendSender struct {
	client   *resend.Client
	fromAddr string
}

// NewResend erstellt einen neuen ResendSender.
func NewResend(apiKey, fromAddr string) *ResendSender {
	return &ResendSender{
		client:   resend.NewClient(apiKey),
		fromAddr: fromAddr,
	}
}

// SendInvite versendet eine Einladungs-E-Mail.
func (s *ResendSender) SendInvite(_ context.Context, toEmail, orgName, inviteToken string) error {
	html := fmt.Sprintf(`<h2>Einladung zu %s auf max-cloud</h2>
<p>Du wurdest eingeladen, der Organisation <strong>%s</strong> auf max-cloud beizutreten.</p>
<p>Nutze den folgenden Befehl, um die Einladung anzunehmen:</p>
<pre>maxcloud auth accept-invite --token %s</pre>
<p>Der Token ist 7 Tage gültig.</p>`,
		orgName, orgName, inviteToken)

	_, err := s.client.Emails.Send(&resend.SendEmailRequest{
		From:    s.fromAddr,
		To:      []string{toEmail},
		Subject: fmt.Sprintf("Einladung zu %s auf max-cloud", orgName),
		Html:    html,
	})
	if err != nil {
		return fmt.Errorf("sending invite email via Resend: %w", err)
	}
	return nil
}
