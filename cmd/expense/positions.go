package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	positionsListPage int
	positionsListAll  bool
)

var positionsCmd = &cobra.Command{
	Use:   "positions",
	Short: "Position operations",
}

var positionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List positions",
	RunE:  runPositionsList,
}

var positionsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show position details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPositionsShow,
}

func init() {
	positionsListCmd.Flags().IntVar(&positionsListPage, "page", 1, "page number")
	positionsListCmd.Flags().BoolVar(&positionsListAll, "all", false, "fetch all pages")
	positionsCmd.AddCommand(positionsListCmd)
	positionsCmd.AddCommand(positionsShowCmd)
}

func runPositionsList(cmd *cobra.Command, args []string) error {
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

	if positionsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.Position, bool, error) {
			return svc.ListPositions(oid, page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"positions": all})
		}
		return f.Format(os.Stdout, all)
	}

	positions, _, err := svc.ListPositions(oid, positionsListPage)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"positions": positions})
	}
	return f.Format(os.Stdout, positions)
}

func runPositionsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	pos, err := svc.GetPosition(oid, args[0])
	if err != nil {
		return err
	}
	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, pos)
}
