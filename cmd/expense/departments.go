package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	deptsListPage    int
	deptsListAll     bool
	deptsListKeyword string
)

var departmentsCmd = &cobra.Command{
	Use:   "departments",
	Short: "Department operations (API: depts)",
}

var deptsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List departments",
	RunE:  runDeptsList,
}

var deptsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show department details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runDeptsShow,
}

func init() {
	deptsListCmd.Flags().IntVar(&deptsListPage, "page", 1, "page number")
	deptsListCmd.Flags().BoolVar(&deptsListAll, "all", false, "fetch all pages")
	deptsListCmd.Flags().StringVar(&deptsListKeyword, "keyword", "", "search keyword (max 50 chars)")
	departmentsCmd.AddCommand(deptsListCmd)
	departmentsCmd.AddCommand(deptsShowCmd)
}

func runDeptsList(cmd *cobra.Command, args []string) error {
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

	if deptsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.Dept, bool, error) {
			return svc.ListDepts(oid, page, deptsListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"depts": all})
		}
		return f.Format(os.Stdout, all)
	}

	depts, _, err := svc.ListDepts(oid, deptsListPage, deptsListKeyword)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"depts": depts})
	}
	return f.Format(os.Stdout, depts)
}

func runDeptsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	dept, err := svc.GetDept(oid, args[0])
	if err != nil {
		return err
	}
	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, dept)
}
