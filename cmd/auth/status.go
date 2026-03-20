package auth

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Show authentication status for all services for the current profile.`,
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, args []string) error {
	profile := cmdutil.GetProfile(cmd)

	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tSTATUS\tEXPIRES\tSCOPE")

	// Sort services for deterministic output.
	names := make([]string, 0, len(api.Services))
	for k := range api.Services {
		names = append(names, k)
	}
	sort.Strings(names)

	for _, name := range names {
		ts, err := tokens.Get(profile, name)
		if err != nil {
			return err
		}

		status := "not logged in"
		expires := "-"
		scope := "-"

		if ts != nil {
			if ts.IsExpired() {
				status = "expired"
			} else {
				status = "active"
			}
			if !ts.ExpiresAt.IsZero() {
				expires = ts.ExpiresAt.Format(time.RFC3339)
			}
			scope = ts.Scope
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, status, expires, scope)
	}

	return w.Flush()
}
