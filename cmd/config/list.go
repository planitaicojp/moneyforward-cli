package config

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration values",
	Long:  `List all configuration values for the current profile.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	profile := getProfile(cmd)

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Build key-value pairs.
	kvs := map[string]string{
		"active_profile": profile,
		"format":         cfg.Defaults.Format,
	}
	if p, ok := cfg.Profiles[profile]; ok {
		if p.ClientID != "" {
			kvs["client_id"] = p.ClientID
		}
	}

	keys := make([]string, 0, len(kvs))
	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE")
	for _, k := range keys {
		fmt.Fprintf(w, "%s\t%s\n", k, kvs[k])
	}
	return w.Flush()
}
