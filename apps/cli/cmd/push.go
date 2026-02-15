package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	pushName string
	pushTag  string
)

var pushCmd = &cobra.Command{
	Use:   "push [image]",
	Short: "Push a Docker image to the maxcloud registry",
	Long: `Push a Docker image to the maxcloud registry.

The image will be tagged and pushed to registry.maxcloud.dev/{org-id}/{name}:{tag}.

Example:
  maxcloud push myapp:latest --name myapp
  maxcloud push nginx:latest --name web --tag v1`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sourceImage := args[0]

		if pushName == "" {
			return fmt.Errorf("--name is required")
		}

		if pushTag == "" {
			pushTag = "latest"
		}

		authInfo, err := client.AuthStatus()
		if err != nil {
			return formatError(err)
		}

		orgID := authInfo.Organization.ID
		targetImage := fmt.Sprintf("registry.maxcloud.dev/%s/%s:%s", orgID, pushName, pushTag)

		scope := fmt.Sprintf("repository:%s/%s:push", orgID, pushName)
		tokenResp, err := client.GetRegistryToken(scope)
		if err != nil {
			return formatError(err)
		}

		tagCmd := exec.Command("docker", "tag", sourceImage, targetImage)
		tagCmd.Stdout = os.Stdout
		tagCmd.Stderr = os.Stderr
		if err := tagCmd.Run(); err != nil {
			return fmt.Errorf("failed to tag image: %w", err)
		}

		authString := base64.StdEncoding.EncodeToString([]byte("oauth2accesstoken:" + tokenResp.Token))
		loginCmd := exec.Command("docker", "login", "registry.maxcloud.dev", "--username", "oauth2accesstoken", "--password-stdin")
		loginCmd.Stdin = strings.NewReader(authString)
		loginCmd.Stdout = os.Stdout
		loginCmd.Stderr = os.Stderr
		if err := loginCmd.Run(); err != nil {
			return fmt.Errorf("failed to login to registry: %w", err)
		}

		pushCmd := exec.Command("docker", "push", targetImage)
		pushCmd.Stdout = os.Stdout
		pushCmd.Stderr = os.Stderr
		if err := pushCmd.Run(); err != nil {
			return fmt.Errorf("failed to push image: %w", err)
		}

		fmt.Printf("\nPushed: %s\n", targetImage)
		fmt.Printf("Deploy with: maxcloud deploy %s --name %s\n", targetImage, pushName)
		return nil
	},
}

func init() {
	pushCmd.Flags().StringVar(&pushName, "name", "", "Image name in registry (required)")
	pushCmd.Flags().StringVar(&pushTag, "tag", "latest", "Image tag")
	pushCmd.MarkFlagRequired("name")
}
