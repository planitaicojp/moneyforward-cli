package expense

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/config"
)

var officeIDFlag string

var ExpenseCmd = &cobra.Command{
	Use:   "expense",
	Short: "Money Forward Cloud Expense",
	Long:  "Commands for Money Forward Cloud Expense API",
}

func init() {
	ExpenseCmd.PersistentFlags().StringVar(&officeIDFlag, "office-id", "", "office ID (auto-detected if only one office)")

	ExpenseCmd.AddCommand(officesCmd)
	ExpenseCmd.AddCommand(departmentsCmd)
	ExpenseCmd.AddCommand(projectsCmd)
	ExpenseCmd.AddCommand(categoriesCmd)
	ExpenseCmd.AddCommand(taxesCmd)
	ExpenseCmd.AddCommand(positionsCmd)
}

func newExpenseService(cmd *cobra.Command) (*api.ExpenseService, error) {
	profile := cmdutil.GetProfile(cmd)
	token, err := cmdutil.EnsureValidToken(profile, api.Services["expense"])
	if err != nil {
		return nil, err
	}
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	version := cmd.Root().Version
	if version == "" {
		version = "dev"
	}
	client := api.NewWithToken(token, version, verbose)
	return api.NewExpenseServiceDefault(client), nil
}

// resolveOfficeID resolves the office ID from:
// 1. --office-id flag
// 2. Profile config (office_id in config.yaml)
// 3. Auto-detect: GET /offices, use if exactly 1, error if multiple.
func resolveOfficeID(cmd *cobra.Command, svc *api.ExpenseService) (string, error) {
	if officeIDFlag != "" {
		return officeIDFlag, nil
	}

	profile := cmdutil.GetProfile(cmd)
	cfg, err := config.Load()
	if err == nil {
		if p, ok := cfg.Profiles[profile]; ok && p.OfficeID != "" {
			return p.OfficeID, nil
		}
	}

	offices, hasNext, err := svc.ListOffices(1)
	if err != nil {
		return "", fmt.Errorf("auto-detecting office: %w", err)
	}

	if len(offices) == 0 {
		return "", fmt.Errorf("no offices found for this account")
	}
	if len(offices) == 1 && !hasNext {
		return offices[0].ID, nil
	}

	var b strings.Builder
	b.WriteString("multiple offices found; specify --office-id:\n")
	for _, o := range offices {
		fmt.Fprintf(&b, "  %s  %s\n", o.ID, o.Name)
	}
	if hasNext {
		b.WriteString("  ... (more offices on next page)\n")
	}
	return "", fmt.Errorf("%s", b.String())
}
