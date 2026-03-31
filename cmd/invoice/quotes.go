package invoice

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/api"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var quotesCmd = &cobra.Command{
	Use:   "quotes",
	Short: "Quote operations",
}

// --- list ---

var (
	quotesListPage      int
	quotesListPerPage   int
	quotesListPartnerID string
	quotesListPartner   string
	quotesListStatus    string
	quotesListFrom      string
	quotesListTo        string
	quotesListQuery     string
	quotesListAll       bool
)

var quotesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quotes",
	RunE:  runQuotesList,
}

// --- show ---

var quotesShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show quote details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesShow,
}

// --- create ---

var (
	quotesCreatePartnerID   string
	quotesCreateQuoteDate   string
	quotesCreateExpiredDate string
	quotesCreateDepartment  string
	quotesCreateTitle       string
	quotesCreateMemo        string
	quotesCreateItemFlags   []string
	quotesCreateItemsFile   string
	quotesCreateItemsStdin  bool
	quotesCreateDryRun      bool
)

var quotesCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a quote",
	RunE:  runQuotesCreate,
}

// --- update ---

var (
	quotesUpdateTitle       string
	quotesUpdateMemo        string
	quotesUpdateQuoteDate   string
	quotesUpdateExpiredDate string
	quotesUpdateItemFlags   []string
	quotesUpdateItemsFile   string
	quotesUpdateItemsStdin  bool
	quotesUpdateDryRun      bool
)

var quotesUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a quote",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesUpdate,
}

// --- delete ---

var quotesDeleteYes bool

var quotesDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a quote",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesDelete,
}

// --- set-status ---

var quotesSetStatusValue string

var quotesSetStatusCmd = &cobra.Command{
	Use:   "set-status <id>",
	Short: "Set quote status",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesSetStatus,
}

// --- to-billing ---

var quotesToBillingCmd = &cobra.Command{
	Use:   "to-billing <id>",
	Short: "Convert quote to billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesToBilling,
}

// --- pdf ---

var (
	quotesPDFDownload bool
	quotesPDFOutput   string
)

var quotesPDFCmd = &cobra.Command{
	Use:   "pdf <id>",
	Short: "Get quote PDF",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runQuotesPDF,
}

func init() {
	quotesListCmd.Flags().IntVar(&quotesListPage, "page", 1, "page number")
	quotesListCmd.Flags().IntVar(&quotesListPerPage, "per-page", 25, "items per page (max 100)")
	quotesListCmd.Flags().StringVar(&quotesListPartnerID, "partner-id", "", "filter by partner ID")
	quotesListCmd.Flags().StringVar(&quotesListPartner, "partner", "", "filter by partner name (resolved to ID)")
	quotesListCmd.Flags().StringVar(&quotesListStatus, "status", "", "filter by status (draft|sent|accepted|rejected|cancelled)")
	quotesListCmd.Flags().StringVar(&quotesListFrom, "from", "", "from date (YYYY-MM-DD)")
	quotesListCmd.Flags().StringVar(&quotesListTo, "to", "", "to date (YYYY-MM-DD)")
	quotesListCmd.Flags().StringVar(&quotesListQuery, "query", "", "search query")
	quotesListCmd.Flags().BoolVar(&quotesListAll, "all", false, "fetch all pages")

	quotesCreateCmd.Flags().StringVar(&quotesCreatePartnerID, "partner-id", "", "partner ID (required if --department-id is omitted)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateQuoteDate, "quote-date", "", "quote date YYYY-MM-DD (required)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateExpiredDate, "expired-date", "", "expiry date YYYY-MM-DD (required)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateDepartment, "department-id", "", "department ID (auto-resolved from --partner-id if omitted)")
	quotesCreateCmd.Flags().StringVar(&quotesCreateTitle, "title", "", "quote title")
	quotesCreateCmd.Flags().StringVar(&quotesCreateMemo, "memo", "", "memo")
	quotesCreateCmd.Flags().StringArrayVar(&quotesCreateItemFlags, "item", nil, `line item: "name=X,price=N,quantity=N,excise=10"`)
	quotesCreateCmd.Flags().StringVar(&quotesCreateItemsFile, "items-file", "", "JSON or YAML file with line items")
	quotesCreateCmd.Flags().BoolVar(&quotesCreateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	quotesCreateCmd.Flags().BoolVar(&quotesCreateDryRun, "dry-run", false, "print request body without sending")
	_ = quotesCreateCmd.MarkFlagRequired("quote-date")
	_ = quotesCreateCmd.MarkFlagRequired("expired-date")

	quotesUpdateCmd.Flags().StringVar(&quotesUpdateTitle, "title", "", "quote title")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateMemo, "memo", "", "memo")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateQuoteDate, "quote-date", "", "quote date YYYY-MM-DD")
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateExpiredDate, "expired-date", "", "expiry date YYYY-MM-DD")
	quotesUpdateCmd.Flags().StringArrayVar(&quotesUpdateItemFlags, "item", nil, `line item: "name=X,price=N,quantity=N,excise=10"`)
	quotesUpdateCmd.Flags().StringVar(&quotesUpdateItemsFile, "items-file", "", "JSON or YAML file with line items")
	quotesUpdateCmd.Flags().BoolVar(&quotesUpdateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	quotesUpdateCmd.Flags().BoolVar(&quotesUpdateDryRun, "dry-run", false, "print request body without sending")

	quotesDeleteCmd.Flags().BoolVar(&quotesDeleteYes, "yes", false, "skip confirmation prompt")

	quotesSetStatusCmd.Flags().StringVar(&quotesSetStatusValue, "status", "", "quote status (draft|sent|accepted|rejected|cancelled) (required)")
	_ = quotesSetStatusCmd.MarkFlagRequired("status")

	quotesPDFCmd.Flags().BoolVar(&quotesPDFDownload, "download", false, "download PDF file")
	quotesPDFCmd.Flags().StringVar(&quotesPDFOutput, "output", "", "output file path")

	quotesCmd.AddCommand(quotesListCmd)
	quotesCmd.AddCommand(quotesShowCmd)
	quotesCmd.AddCommand(quotesCreateCmd)
	quotesCmd.AddCommand(quotesUpdateCmd)
	quotesCmd.AddCommand(quotesDeleteCmd)
	quotesCmd.AddCommand(quotesSetStatusCmd)
	quotesCmd.AddCommand(quotesToBillingCmd)
	quotesCmd.AddCommand(quotesPDFCmd)
}

func runQuotesList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	partnerID := quotesListPartnerID
	if quotesListPartner != "" {
		partnerID, err = resolvePartnerID(svc, quotesListPartner)
		if err != nil {
			return err
		}
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if quotesListAll {
		allQuotes, err := fetchAll(func(page int) ([]model.Quote, *pagination.Result, error) {
			opts := api.QuoteListOptions{
				Params:    pagination.Params{Page: page, PerPage: 100},
				PartnerID: partnerID,
				Status:    quotesListStatus,
				From:      quotesListFrom,
				To:        quotesListTo,
				Query:     quotesListQuery,
			}
			return svc.ListQuotes(opts)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allQuotes})
		}
		return f.Format(os.Stdout, allQuotes)
	}

	opts := api.QuoteListOptions{
		Params:    pagination.Params{Page: quotesListPage, PerPage: quotesListPerPage},
		PartnerID: partnerID,
		Status:    quotesListStatus,
		From:      quotesListFrom,
		To:        quotesListTo,
		Query:     quotesListQuery,
	}
	quotes, pg, err := svc.ListQuotes(opts)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": quotes, "pagination": pg})
	}
	return f.Format(os.Stdout, quotes)
}

func runQuotesShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	quote, err := svc.GetQuote(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	items, err := resolveLineItems(quotesCreateItemFlags, quotesCreateItemsFile, quotesCreateItemsStdin)
	if err != nil {
		return err
	}

	departmentID := quotesCreateDepartment
	if departmentID == "" {
		if quotesCreatePartnerID == "" {
			return fmt.Errorf("either --department-id or --partner-id must be provided")
		}
		depts, err := svc.ListPartnerDepartments(quotesCreatePartnerID)
		if err != nil {
			return fmt.Errorf("resolving department: %w", err)
		}
		if len(depts) == 0 {
			return fmt.Errorf("partner has no departments registered; use --department-id")
		}
		if len(depts) > 1 {
			return fmt.Errorf("multiple departments found for partner; use --department-id to specify one")
		}
		departmentID = depts[0].ID
	}

	params := model.CreateQuoteParams{
		DepartmentID: departmentID,
		QuoteDate:    quotesCreateQuoteDate,
		ExpiredDate:  quotesCreateExpiredDate,
		Title:        quotesCreateTitle,
		Memo:         quotesCreateMemo,
		Items:        items,
	}

	if quotesCreateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	quote, err := svc.CreateQuote(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateQuoteParams
	if cmd.Flags().Changed("title") {
		params.Title = &quotesUpdateTitle
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &quotesUpdateMemo
	}
	if cmd.Flags().Changed("quote-date") {
		params.QuoteDate = &quotesUpdateQuoteDate
	}
	if cmd.Flags().Changed("expired-date") {
		params.ExpiredDate = &quotesUpdateExpiredDate
	}

	items, err := resolveLineItems(quotesUpdateItemFlags, quotesUpdateItemsFile, quotesUpdateItemsStdin)
	if err != nil {
		return err
	}
	if items != nil {
		params.Items = items
	}

	if quotesUpdateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	quote, err := svc.UpdateQuote(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesDelete(cmd *cobra.Command, args []string) error {
	if !quotesDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete quote %s?", args[0]))
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

	return svc.DeleteQuote(args[0])
}

func runQuotesSetStatus(cmd *cobra.Command, args []string) error {
	switch quotesSetStatusValue {
	case string(model.QuoteStatusDraft),
		string(model.QuoteStatusSent),
		string(model.QuoteStatusAccepted),
		string(model.QuoteStatusRejected),
		string(model.QuoteStatusCancelled):
	default:
		return fmt.Errorf("invalid quote status %q: must be draft, sent, accepted, rejected, or cancelled", quotesSetStatusValue)
	}

	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	quote, err := svc.SetQuoteStatus(args[0], model.QuoteStatus(quotesSetStatusValue))
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, quote)
}

func runQuotesToBilling(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.ConvertQuoteToBilling(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runQuotesPDF(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	pdfURL, err := svc.GetQuotePDF(args[0])
	if err != nil {
		return err
	}
	if pdfURL == "" {
		return fmt.Errorf("quote %s has no PDF URL available", args[0])
	}

	if !quotesPDFDownload && quotesPDFOutput == "" {
		fmt.Println(pdfURL)
		return nil
	}

	resp, err := svc.DownloadPDF(pdfURL)
	if err != nil {
		return fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading PDF: unexpected status %d", resp.StatusCode)
	}

	outPath := quotesPDFOutput
	if outPath == "" {
		outPath = args[0] + ".pdf"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}

	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()

	if copyErr != nil {
		_ = os.Remove(outPath)
		return fmt.Errorf("writing PDF: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("closing PDF file: %w", closeErr)
	}

	fmt.Fprintf(os.Stderr, "Downloaded to %s\n", outPath)
	return nil
}
