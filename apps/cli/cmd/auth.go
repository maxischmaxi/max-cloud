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
	registerEmail   string
	registerOrgName string
	apiKeyName      string
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication and account management",
}

var authRegisterCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new account",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := client.Register(models.RegisterRequest{
			Email:   registerEmail,
			OrgName: registerOrgName,
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

		// Token für nachfolgende Befehle setzen
		client.Token = resp.APIKey

		fmt.Printf("Registration successful!\n")
		fmt.Printf("  Email:        %s\n", resp.User.Email)
		fmt.Printf("  Organization: %s\n", resp.Organization.Name)
		fmt.Printf("  API Key:      %s\n", resp.APIKey)
		fmt.Printf("\nCredentials saved. You can now use maxcloud commands.\n")

		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := client.AuthStatus()
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("Authenticated as:\n")
		fmt.Printf("  Email:        %s\n", info.User.Email)
		fmt.Printf("  Organization: %s\n", info.Organization.Name)
		fmt.Printf("  Role:         %s\n", info.Role)

		return nil
	},
}

var apiKeyCmd = &cobra.Command{
	Use:   "api-key",
	Short: "Manage API keys",
}

var apiKeyCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		resp, err := client.CreateAPIKey(models.CreateAPIKeyRequest{
			Name: apiKeyName,
		})
		if err != nil {
			return formatError(err)
		}

		fmt.Printf("API key created:\n")
		fmt.Printf("  Name:   %s\n", resp.Info.Name)
		fmt.Printf("  Key:    %s\n", resp.APIKey)
		fmt.Printf("\nSave this key — it won't be shown again.\n")

		return nil
	},
}

var apiKeyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all API keys",
	RunE: func(cmd *cobra.Command, args []string) error {
		keys, err := client.ListAPIKeys()
		if err != nil {
			return formatError(err)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tPREFIX\tCREATED\tLAST USED")
		for _, k := range keys {
			lastUsed := "-"
			if k.LastUsedAt != nil {
				lastUsed = k.LastUsedAt.Format(time.DateTime)
			}
			fmt.Fprintf(w, "%s\t%s\tmc_%s...\t%s\t%s\n",
				k.ID, k.Name, k.Prefix,
				k.CreatedAt.Format(time.DateTime),
				lastUsed,
			)
		}
		w.Flush()

		return nil
	},
}

var apiKeyDeleteCmd = &cobra.Command{
	Use:   "delete [key-id]",
	Short: "Delete an API key",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := client.DeleteAPIKey(args[0]); err != nil {
			return formatError(err)
		}
		fmt.Println("API key deleted.")
		return nil
	},
}

func init() {
	authRegisterCmd.Flags().StringVar(&registerEmail, "email", "", "Email address")
	authRegisterCmd.Flags().StringVar(&registerOrgName, "org", "", "Organization name")
	authRegisterCmd.MarkFlagRequired("email")
	authRegisterCmd.MarkFlagRequired("org")

	apiKeyCreateCmd.Flags().StringVar(&apiKeyName, "name", "", "Name for the API key")
	apiKeyCreateCmd.MarkFlagRequired("name")

	apiKeyCmd.AddCommand(apiKeyCreateCmd)
	apiKeyCmd.AddCommand(apiKeyListCmd)
	apiKeyCmd.AddCommand(apiKeyDeleteCmd)

	authCmd.AddCommand(authRegisterCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(apiKeyCmd)
}
