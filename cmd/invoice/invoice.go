package invoice

import "github.com/spf13/cobra"

var InvoiceCmd = &cobra.Command{
	Use:   "invoice",
	Short: "Money Forward Cloud Invoice",
	Long:  "Commands for Money Forward Cloud Invoice API",
}

func init() {
	InvoiceCmd.AddCommand(officeCmd)
	InvoiceCmd.AddCommand(partnersCmd)
	partnersCmd.AddCommand(partnersDepartmentsCmd)
}
