package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	officesListPage int
	officesListAll  bool
)

var officesCmd = &cobra.Command{
	Use:   "offices",
	Short: "Office operations",
}

var officesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List offices",
	RunE:  runOfficesList,
}

func init() {
	officesListCmd.Flags().IntVar(&officesListPage, "page", 1, "page number")
	officesListCmd.Flags().BoolVar(&officesListAll, "all", false, "fetch all pages")
	officesCmd.AddCommand(officesListCmd)
}

func runOfficesList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if officesListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseOffice, bool, error) {
			return svc.ListOffices(page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"offices": all})
		}
		return f.Format(os.Stdout, all)
	}

	offices, _, err := svc.ListOffices(officesListPage)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"offices": offices})
	}
	return f.Format(os.Stdout, offices)
}
