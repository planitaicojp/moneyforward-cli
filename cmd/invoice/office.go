package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var officeCmd = &cobra.Command{
	Use:   "office",
	Short: "Office operations",
}

var officeShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show office information",
	RunE:  runOfficeShow,
}

func init() {
	officeCmd.AddCommand(officeShowCmd)
}

func runOfficeShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	office, err := svc.GetOffice()
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, office)
}
