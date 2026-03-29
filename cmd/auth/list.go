package auth

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles",
	Long:  `List all profiles and their configured services.`,
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	tokens, err := config.LoadTokens()
	if err != nil {
		return err
	}

	// Collect all profile names from both config and tokens.
	profileSet := make(map[string]struct{})
	for name := range cfg.Profiles {
		profileSet[name] = struct{}{}
	}
	for name := range tokens.Profiles {
		profileSet[name] = struct{}{}
	}

	names := make([]string, 0, len(profileSet))
	for name := range profileSet {
		names = append(names, name)
	}
	sort.Strings(names)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROFILE\tACTIVE\tSERVICES")

	activeProfile := cmdutil.GetProfile(cmd)

	for _, name := range names {
		active := ""
		if name == activeProfile {
			active = "*"
		}
		services := tokens.ListServices(name)
		sort.Strings(services)
		fmt.Fprintf(w, "%s\t%s\t%s\n", name, active, strings.Join(services, ", "))
	}

	return w.Flush()
}
