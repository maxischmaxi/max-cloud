package cmd

import (
	"fmt"
	"strings"

	"github.com/max-cloud/shared/pkg/models"
	"github.com/spf13/cobra"
)

var (
	deployName string
	deployEnv  []string
)

var deployCmd = &cobra.Command{
	Use:   "deploy [image]",
	Short: "Deploy a container image as a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		image := args[0]

		envVars := make(map[string]string)
		for _, e := range deployEnv {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid env format %q, expected KEY=VALUE", e)
			}
			envVars[parts[0]] = parts[1]
		}

		req := models.DeployRequest{
			Name:    deployName,
			Image:   image,
			EnvVars: envVars,
		}

		svc, err := client.Deploy(req)
		if err != nil {
			return fmt.Errorf("deploy failed: %w", err)
		}

		fmt.Printf("Service deployed successfully!\n")
		fmt.Printf("  ID:     %s\n", svc.ID)
		fmt.Printf("  Name:   %s\n", svc.Name)
		fmt.Printf("  Image:  %s\n", svc.Image)
		fmt.Printf("  Status: %s\n", svc.Status)
		fmt.Printf("  URL:    %s\n", svc.URL)
		return nil
	},
}

func init() {
	deployCmd.Flags().StringVar(&deployName, "name", "", "Service name (required)")
	deployCmd.MarkFlagRequired("name")
	deployCmd.Flags().StringArrayVar(&deployEnv, "env", nil, "Environment variables (KEY=VALUE, repeatable)")
}
