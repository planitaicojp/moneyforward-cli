package invoice

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

// --- shared helpers ---

func newInvoiceService(cmd *cobra.Command) (*api.InvoiceService, error) {
	profile := cmdutil.GetProfile(cmd)
	token, err := cmdutil.EnsureValidToken(profile, api.Services["invoice"])
	if err != nil {
		return nil, err
	}
	verbose, _ := cmd.Root().PersistentFlags().GetBool("verbose")
	client := api.NewWithToken(token, "dev", verbose)
	return api.NewInvoiceServiceDefault(client), nil
}

func getFormat(cmd *cobra.Command) string {
	format, _ := cmd.Root().PersistentFlags().GetString("format")
	if format == "" {
		format = "table"
	}
	return format
}

// --- partners root ---

var partnersCmd = &cobra.Command{
	Use:   "partners",
	Short: "Partner operations",
}

// --- list ---

var (
	partnersListPage    int
	partnersListPerPage int
	partnersListQuery string
)

var partnersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List partners",
	RunE:  runPartnersList,
}

// --- show ---

var partnersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show partner details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersShow,
}

// --- create ---

var (
	partnersCreateName       string
	partnersCreateNameKana   string
	partnersCreateNameSuffix string
	partnersCreateCode       string
	partnersCreateMemo       string
)

var partnersCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a partner",
	RunE:  runPartnersCreate,
}

// --- update ---

var (
	partnersUpdateName       string
	partnersUpdateNameKana   string
	partnersUpdateNameSuffix string
	partnersUpdateCode       string
	partnersUpdateMemo       string
)

var partnersUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersUpdate,
}

// --- delete ---

var partnersDeleteYes bool

var partnersDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a partner",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runPartnersDelete,
}

func init() {
	partnersListCmd.Flags().IntVar(&partnersListPage, "page", 1, "page number")
	partnersListCmd.Flags().IntVar(&partnersListPerPage, "per-page", 25, "items per page (max 100)")
	partnersListCmd.Flags().StringVar(&partnersListQuery, "query", "", "search query")

	partnersCreateCmd.Flags().StringVar(&partnersCreateName, "name", "", "partner name (required)")
	partnersCreateCmd.Flags().StringVar(&partnersCreateNameKana, "name-kana", "", "partner name in kana")
	partnersCreateCmd.Flags().StringVar(&partnersCreateNameSuffix, "name-suffix", "", "name suffix (e.g. 様, 御中)")
	partnersCreateCmd.Flags().StringVar(&partnersCreateCode, "code", "", "partner code")
	partnersCreateCmd.Flags().StringVar(&partnersCreateMemo, "memo", "", "memo")
	_ = partnersCreateCmd.MarkFlagRequired("name")

	partnersUpdateCmd.Flags().StringVar(&partnersUpdateName, "name", "", "partner name")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateNameKana, "name-kana", "", "partner name in kana")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateNameSuffix, "name-suffix", "", "name suffix")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateCode, "code", "", "partner code")
	partnersUpdateCmd.Flags().StringVar(&partnersUpdateMemo, "memo", "", "memo")

	partnersDeleteCmd.Flags().BoolVar(&partnersDeleteYes, "yes", false, "skip confirmation prompt")

	partnersCmd.AddCommand(partnersListCmd)
	partnersCmd.AddCommand(partnersShowCmd)
	partnersCmd.AddCommand(partnersCreateCmd)
	partnersCmd.AddCommand(partnersUpdateCmd)
	partnersCmd.AddCommand(partnersDeleteCmd)
}

func runPartnersList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := pagination.Params{
		Page:    partnersListPage,
		PerPage: partnersListPerPage,
		Query:   partnersListQuery,
	}

	partners, pg, err := svc.ListPartners(params)
	if err != nil {
		return err
	}

	format := getFormat(cmd)
	f := output.New(format)

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": partners, "pagination": pg})
	}
	return f.Format(os.Stdout, partners)
}

func runPartnersShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	partner, err := svc.GetPartner(args[0])
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}

func runPartnersCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := model.CreatePartnerParams{
		Name:       partnersCreateName,
		NameKana:   partnersCreateNameKana,
		NameSuffix: partnersCreateNameSuffix,
		Code:       partnersCreateCode,
		Memo:       partnersCreateMemo,
	}

	partner, err := svc.CreatePartner(params)
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}

func runPartnersUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdatePartnerParams
	if cmd.Flags().Changed("name") {
		params.Name = &partnersUpdateName
	}
	if cmd.Flags().Changed("name-kana") {
		params.NameKana = &partnersUpdateNameKana
	}
	if cmd.Flags().Changed("name-suffix") {
		params.NameSuffix = &partnersUpdateNameSuffix
	}
	if cmd.Flags().Changed("code") {
		params.Code = &partnersUpdateCode
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &partnersUpdateMemo
	}

	partner, err := svc.UpdatePartner(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(getFormat(cmd))
	return f.Format(os.Stdout, partner)
}

func runPartnersDelete(cmd *cobra.Command, args []string) error {
	if !partnersDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete partner %s?", args[0]))
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
	}

	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	return svc.DeletePartner(args[0])
}
