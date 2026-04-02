package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	categoriesListPage    int
	categoriesListAll     bool
	categoriesListKeyword string
)

var categoriesCmd = &cobra.Command{
	Use:   "categories",
	Short: "Expense category operations (API: ex_items)",
}

var categoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List expense categories",
	RunE:  runCategoriesList,
}

var categoriesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show expense category details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runCategoriesShow,
}

func init() {
	categoriesListCmd.Flags().IntVar(&categoriesListPage, "page", 1, "page number")
	categoriesListCmd.Flags().BoolVar(&categoriesListAll, "all", false, "fetch all pages")
	categoriesListCmd.Flags().StringVar(&categoriesListKeyword, "keyword", "", "search keyword (max 50 chars)")
	categoriesCmd.AddCommand(categoriesListCmd)
	categoriesCmd.AddCommand(categoriesShowCmd)
}

func runCategoriesList(cmd *cobra.Command, args []string) error {
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

	if categoriesListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExItem, bool, error) {
			return svc.ListExItems(oid, page, categoriesListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"ex_items": all})
		}
		return f.Format(os.Stdout, all)
	}

	items, _, err := svc.ListExItems(oid, categoriesListPage, categoriesListKeyword)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"ex_items": items})
	}
	return f.Format(os.Stdout, items)
}

func runCategoriesShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	item, err := svc.GetExItem(oid, args[0])
	if err != nil {
		return err
	}
	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}
