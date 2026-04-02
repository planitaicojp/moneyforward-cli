package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	taxesListPage int
	taxesListAll  bool
)

var taxesCmd = &cobra.Command{
	Use:   "taxes",
	Short: "Tax classification operations (API: excises)",
}

var taxesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tax classifications",
	RunE:  runTaxesList,
}

var taxesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show tax classification details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runTaxesShow,
}

func init() {
	taxesListCmd.Flags().IntVar(&taxesListPage, "page", 1, "page number")
	taxesListCmd.Flags().BoolVar(&taxesListAll, "all", false, "fetch all pages")
	taxesCmd.AddCommand(taxesListCmd)
	taxesCmd.AddCommand(taxesShowCmd)
}

func runTaxesList(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if taxesListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseExcise, bool, error) {
			return svc.ListExcises(oid, page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"excises": all})
		}
		return f.Format(os.Stdout, all)
	}

	excises, _, err := svc.ListExcises(oid, taxesListPage)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"excises": excises})
	}
	return f.Format(os.Stdout, excises)
}

func runTaxesShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	excise, err := svc.GetExcise(oid, args[0])
	if err != nil {
		return err
	}
	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, excise)
}
