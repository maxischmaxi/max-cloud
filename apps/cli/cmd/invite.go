package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/max-cloud/shared/pkg/models"
	"github.com/spf13/cobra"
)

var (
	inviteEmail string
	inviteRole  string
	inviteToken string
)

var inviteCmd = &cobra.Command{
	Use:   "invite",
	Short: "Manage organization invitations",
}

var inviteCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Invite a user to the organization",
	RunE: func(cmd *cobra.Command, args []string) error {
		role := models.OrgRole(inviteRole)
		resp, err := client.CreateInvite(models.InviteRequest{
			Email: inviteEmail,
			Role:  role,
		})
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Invitation created:\n")
		fmt.Printf("  ID:      %s\n", resp.Invitation.ID)
		fmt.Printf("  Email:   %s\n", resp.Invitation.Email)
		fmt.Printf("  Role:    %s\n", resp.Invitation.Role)
		fmt.Printf("  Org:     %s\n", resp.Invitation.OrgName)
		fmt.Printf("  Expires: %s\n", resp.Invitation.ExpiresAt.Format(time.DateTime))

		if resp.Token != "" {
			fmt.Printf("\n  Token:   %s\n", resp.Token)
			fmt.Printf("\n  Accept command:\n")
			fmt.Printf("    maxcloud auth accept-invite --token %s\n", resp.Token)
		}

		return nil
	},
}

var inviteListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pending invitations",
	RunE: func(cmd *cobra.Command, args []string) error {
		invites, err := client.ListInvites()
		if err != nil {
			return formatError(err)
		}

		if len(invites) == 0 {
			fmt.Println("No pending invitations.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tEMAIL\tROLE\tEXPIRES")
		for _, inv := range invites {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n",
				inv.ID, inv.Email, inv.Role,
				inv.ExpiresAt.Format(time.DateTime),
			)
		}
		w.Flush()

		return nil
	},
}

var inviteRevokeCmd = &cobra.Command{
	Use:   "revoke [invite-id]",
	Short: "Revoke a pending invitation",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.RevokeInvite(args[0]); err != nil {
			return formatError(err)
		}
		fmt.Println("Invitation revoked.")
		return nil
	},
}

var acceptInviteCmd = &cobra.Command{
	Use:   "accept-invite",
	Short: "Accept an organization invitation",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := client.AcceptInvite(models.AcceptInviteRequest{
			Token: inviteToken,
		})
		if err != nil {
			return formatError(err)
		}

		// Credentials speichern
		if err := saveCredentials(&Credentials{
			APIURL: apiURL,
			APIKey: resp.APIKey,
		}); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not save credentials: %v\n", err)
		}

		fmt.Printf("Invitation accepted!\n")
		fmt.Printf("  Email:        %s\n", resp.User.Email)
		fmt.Printf("  Organization: %s\n", resp.Organization.Name)
		fmt.Printf("  Role:         %s\n", resp.Role)
		fmt.Printf("  API Key:      %s\n", resp.APIKey)
		fmt.Printf("\nCredentials saved. You can now use maxcloud commands.\n")

		return nil
	},
}

func init() {
	inviteCreateCmd.Flags().StringVar(&inviteEmail, "email", "", "Email address to invite")
	inviteCreateCmd.Flags().StringVar(&inviteRole, "role", "member", "Role for the invited user (member or admin)")
	inviteCreateCmd.MarkFlagRequired("email")

	acceptInviteCmd.Flags().StringVar(&inviteToken, "token", "", "Invitation token")
	acceptInviteCmd.MarkFlagRequired("token")

	inviteCmd.AddCommand(inviteCreateCmd)
	inviteCmd.AddCommand(inviteListCmd)
	inviteCmd.AddCommand(inviteRevokeCmd)

	authCmd.AddCommand(inviteCmd)
	authCmd.AddCommand(acceptInviteCmd)
}
