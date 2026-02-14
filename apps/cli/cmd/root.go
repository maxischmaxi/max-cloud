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
	client  *api.Client
)

var rootCmd = &cobra.Command{
	Use:   "maxcloud",
	Short: "maxcloud - Deutsche Cloud Run Alternative",
	Long:  "CLI for managing serverless containers on max-cloud.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		client = api.NewClient(apiURL)
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", "http://localhost:8080", "API server URL")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(deployCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(deleteCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(os.Stdout, "maxcloud version %s\n", version)
	},
}
