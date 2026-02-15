package email

import "context"

// MockSender ist ein Sender für Tests.
type MockSender struct {
	SendInviteFunc func(ctx context.Context, toEmail, orgName, inviteToken string) error
	LastInvite     InviteCall
}

// InviteCall speichert den letzten Aufruf von SendInvite.
type InviteCall struct {
	ToEmail     string
	OrgName     string
	InviteToken string
}

// NewMock erstellt einen neuen MockSender.
func NewMock() *MockSender {
	return &MockSender{
		SendInviteFunc: func(ctx context.Context, toEmail, orgName, inviteToken string) error {
			return nil
		},
	}
}

// SendInvite ruft SendInviteFunc auf und speichert den Aufruf.
func (m *MockSender) SendInvite(ctx context.Context, toEmail, orgName, inviteToken string) error {
	m.LastInvite = InviteCall{
		ToEmail:     toEmail,
		OrgName:     orgName,
		InviteToken: inviteToken,
	}
	if m.SendInviteFunc != nil {
		return m.SendInviteFunc(ctx, toEmail, orgName, inviteToken)
	}
	return nil
}
