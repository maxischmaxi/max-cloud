package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all deployed services",
	RunE: func(cmd *cobra.Command, args []string) error {
		services, err := client.ListServices()
		if err != nil {
			return formatError(err)
		}

		if len(services) == 0 {
			fmt.Println("No services deployed.")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tIMAGE\tSTATUS\tURL")
		for _, svc := range services {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", svc.Name, svc.Image, svc.Status, svc.URL)
		}
		w.Flush()
		return nil
	},
}
