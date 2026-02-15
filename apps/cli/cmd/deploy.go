package cmd

import (
	"fmt"
	"strings"

	"github.com/max-cloud/shared/pkg/models"
	"github.com/spf13/cobra"
)

var (
	deployName    string
	deployEnv     []string
	deployPort    int
	deployCommand string
	deployArgs    string
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
			Port:    deployPort,
			Command: parseCSV(deployCommand),
			Args:    parseCSV(deployArgs),
			EnvVars: envVars,
		}

		svc, err := client.Deploy(req)
		if err != nil {
			return formatError(err)
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

func parseCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func init() {
	deployCmd.Flags().StringVar(&deployName, "name", "", "Service name (required)")
	deployCmd.MarkFlagRequired("name")
	deployCmd.Flags().StringArrayVar(&deployEnv, "env", nil, "Environment variables (KEY=VALUE, repeatable)")
	deployCmd.Flags().IntVar(&deployPort, "port", 0, "Container port (0 = auto-detect from EXPOSE)")
	deployCmd.Flags().StringVar(&deployCommand, "command", "", "Override ENTRYPOINT (comma-separated: python,app.py)")
	deployCmd.Flags().StringVar(&deployArgs, "args", "", "Override CMD (comma-separated: --port,3000)")
}
