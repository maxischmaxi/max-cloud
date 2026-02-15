package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images in your registry",
	Long: `List all images in your organization's registry namespace.

Images are stored at registry.maxcloud.dev/{org-id}/{name}:{tag}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		authInfo, err := client.AuthStatus()
		if err != nil {
			return formatError(err)
		}

		_, _ = fmt.Printf("Registry: registry.maxcloud.dev/%s/\n\n", authInfo.Organization.ID)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTAG\tSTATUS")
		fmt.Fprintln(w, "----\t---\t------")
		fmt.Fprintln(w, "(no images pushed yet)")
		w.Flush()

		fmt.Printf("\nPush an image with: maxcloud push <source-image> --name <name>\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(imagesCmd)
}
