package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [service-id]",
	Short: "Delete a deployed service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]

		if err := client.DeleteService(id); err != nil {
			return formatError(err)
		}

		fmt.Printf("Service %s deleted.\n", id)
		return nil
	},
}
