package cmd

import (
	"fmt"
	"os"

	"github.com/max-cloud/shared/pkg/api"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	apiURL  string
	apiKey  string
	client  *api.Client
)

var rootCmd = &cobra.Command{
	Use:           "maxcloud",
	Short:         "maxcloud - Deutsche Cloud Run Alternative",
	Long:          "CLI for managing serverless containers on max-cloud.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client = api.NewClient(apiURL)

		// Token-Priorität: --api-key Flag > MAXCLOUD_API_KEY env > ~/.config/maxcloud/credentials
		token := apiKey
		if token == "" {
			token = os.Getenv("MAXCLOUD_API_KEY")
		}
		if token == "" {
			if creds, err := loadCredentials(); err == nil && creds != nil {
				token = creds.APIKey
				if creds.APIURL != "" && !cmd.Flags().Changed("api-url") {
					client.BaseURL = creds.APIURL
				}
			}
		}
		client.Token = token
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(pushCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stdout, "maxcloud version %s\n", version)
	},
}
