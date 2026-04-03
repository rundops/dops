package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"dops/internal/domain"
	"dops/internal/history"

	"github.com/spf13/cobra"
)

func newHistoryCmd(dopsDir string) *cobra.Command {
	var runbookID string
	var status string
	var limit int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show execution history",
		Long:  "List recent runbook executions with status, duration, and exit code.",
		RunE: func(cmd *cobra.Command, args []string) error {
			historyDir := filepath.Join(dopsDir, "history")
			store := history.NewFileStore(historyDir, 0)

			opts := history.ListOpts{
				RunbookID: runbookID,
				Status:    domain.ExecStatus(status),
				Limit:     limit,
			}

			records, err := store.List(opts)
			if err != nil {
				return fmt.Errorf("list history: %w", err)
			}

			if len(records) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No executions found.")
				return nil
			}

			// Table header.
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%-20s  %-28s  %-10s  %-10s  %-4s\n",
				"TIME", "RUNBOOK", "STATUS", "DURATION", "EXIT")
			fmt.Fprintf(out, "%s\n", strings.Repeat("─", 78))

			for _, r := range records {
				ts := r.StartTime.Format("2006-01-02 15:04:05")
				rbID := r.RunbookID
				if len(rbID) > 28 {
					rbID = rbID[:25] + "..."
				}
				dur := r.Duration
				if dur == "" {
					dur = "-"
				}
				fmt.Fprintf(out, "%-20s  %-28s  %-10s  %-10s  %-4d\n",
					ts, rbID, r.Status, dur, r.ExitCode)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&runbookID, "runbook", "", "Filter by runbook ID")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (success, failed, cancelled)")
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of records to show")

	return cmd
}
