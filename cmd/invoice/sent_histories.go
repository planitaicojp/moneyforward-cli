package invoice

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
)

var sentHistoriesCmd = &cobra.Command{
	Use:   "sent-histories",
	Short: "Sent history operations",
}

var (
	sentHistoriesListPage    int
	sentHistoriesListPerPage int
	sentHistoriesListAll     bool
)

var sentHistoriesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List sent histories",
	RunE:  runSentHistoriesList,
}

func init() {
	sentHistoriesListCmd.Flags().IntVar(&sentHistoriesListPage, "page", 1, "page number")
	sentHistoriesListCmd.Flags().IntVar(&sentHistoriesListPerPage, "per-page", 25, "items per page (max 100)")
	sentHistoriesListCmd.Flags().BoolVar(&sentHistoriesListAll, "all", false, "fetch all pages")

	sentHistoriesCmd.AddCommand(sentHistoriesListCmd)
}

func runSentHistoriesList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if sentHistoriesListAll {
		allHistories, err := cmdutil.FetchAll(func(page int) ([]model.SentHistory, *pagination.Result, error) {
			return svc.ListSentHistories(pagination.Params{Page: page, PerPage: 100})
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allHistories})
		}
		return f.Format(os.Stdout, allHistories)
	}

	params := pagination.Params{Page: sentHistoriesListPage, PerPage: sentHistoriesListPerPage}
	histories, pg, err := svc.ListSentHistories(params)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": histories, "pagination": pg})
	}
	return f.Format(os.Stdout, histories)
}
