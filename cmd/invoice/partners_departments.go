package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var partnersDepartmentsCmd = &cobra.Command{
	Use:   "departments",
	Short: "Partner department operations",
}

var partnersDepartmentsListCmd = &cobra.Command{
	Use:   "list <partner-id>",
	Short: "List departments for a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersDepartmentsList,
}

func init() {
	partnersDepartmentsCmd.AddCommand(partnersDepartmentsListCmd)
}

func runPartnersDepartmentsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	departments, err := svc.ListPartnerDepartments(args[0])
	if err != nil {
		return err
	}

	format := getFormat(cmd)
	f := output.New(format)

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": departments})
	}
	return f.Format(os.Stdout, departments)
}
