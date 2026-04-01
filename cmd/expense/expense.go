package expense

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
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

func resolveOfficeID(cmd *cobra.Command, svc *api.ExpenseService) (string, error) {
	if officeIDFlag != "" {
		return officeIDFlag, nil
	}
	offices, _, err := svc.ListOffices(1)
	if err != nil {
		return "", fmt.Errorf("auto-detecting office: %w", err)
	}
	switch len(offices) {
	case 0:
		return "", fmt.Errorf("no offices found for this account")
	case 1:
		return offices[0].ID, nil
	default:
		msg := "multiple offices found; specify --office-id:\n"
		for _, o := range offices {
			msg += fmt.Sprintf("  %s  %s\n", o.ID, o.Name)
		}
		return "", fmt.Errorf("%s", msg)
	}
}
