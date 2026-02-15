package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   int
)

var logsCmd = &cobra.Command{
	Use:   "logs [service-name]",
	Short: "Show logs for a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serviceName := args[0]

		// Service-Name zu ID auflösen
		services, err := client.ListServices()
		if err != nil {
			return formatError(err)
		}

		var serviceID string
		for _, svc := range services {
			if svc.Name == serviceName {
				serviceID = svc.ID
				break
			}
		}
		if serviceID == "" {
			return fmt.Errorf("service %q not found", serviceName)
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()

		ls, err := client.StreamLogs(ctx, serviceID, logsFollow, logsTail)
		if err != nil {
			return formatError(err)
		}
		defer ls.Close()

		for entry := range ls.Events {
			ts := entry.Timestamp.Format(time.DateTime)
			fmt.Printf("%s [%s] %s\n", ts, entry.Stream, entry.Message)
		}

		return nil
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "Number of lines to show from the end")
}
