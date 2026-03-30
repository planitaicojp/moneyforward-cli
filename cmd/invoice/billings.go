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

var billingsCmd = &cobra.Command{
	Use:   "billings",
	Short: "Billing operations",
}

// --- list ---

var (
	billingsListPage          int
	billingsListPerPage       int
	billingsListPartnerID     string
	billingsListPartner       string
	billingsListPaymentStatus string
	billingsListFrom          string
	billingsListTo            string
	billingsListQuery         string
	billingsListAll           bool
)

var billingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List billings",
	RunE:  runBillingsList,
}

// --- show ---

var billingsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show billing details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsShow,
}

// --- create ---

var (
	billingsCreatePartnerID   string
	billingsCreateBillingDate string
	billingsCreateDepartment  string
	billingsCreateTitle       string
	billingsCreateMemo        string
	billingsCreatePaymentCond string
	billingsCreateDueDate     string
	billingsCreateSalesDate   string
	billingsCreateItemFlags   []string
	billingsCreateItemsFile   string
	billingsCreateItemsStdin  bool
	billingsCreateDryRun      bool
)

var billingsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a billing",
	RunE:  runBillingsCreate,
}

// --- update ---

var (
	billingsUpdateTitle       string
	billingsUpdateMemo        string
	billingsUpdatePaymentCond string
	billingsUpdateBillingDate string
	billingsUpdateDueDate     string
	billingsUpdateSalesDate   string
	billingsUpdateItemsFile   string
	billingsUpdateItemsStdin  bool
	billingsUpdateDryRun      bool
)

var billingsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsUpdate,
}

// --- delete ---

var billingsDeleteYes bool

var billingsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a billing",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsDelete,
}

// --- set-payment-status ---

var billingsSetStatusValue string

var billingsSetPaymentStatusCmd = &cobra.Command{
	Use:   "set-payment-status <id>",
	Short: "Set billing payment status",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsSetPaymentStatus,
}

// --- pdf ---

var (
	billingsPDFDownload bool
	billingsPDFOutput   string
)

var billingsPDFCmd = &cobra.Command{
	Use:   "pdf <id>",
	Short: "Get billing PDF",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runBillingsPDF,
}

func init() {
	billingsListCmd.Flags().IntVar(&billingsListPage, "page", 1, "page number")
	billingsListCmd.Flags().IntVar(&billingsListPerPage, "per-page", 25, "items per page (max 100)")
	billingsListCmd.Flags().StringVar(&billingsListPartnerID, "partner-id", "", "filter by partner ID")
	billingsListCmd.Flags().StringVar(&billingsListPartner, "partner", "", "filter by partner name (resolved to ID)")
	billingsListCmd.Flags().StringVar(&billingsListPaymentStatus, "payment-status", "", "filter by payment status (unsettled|settled)")
	billingsListCmd.Flags().StringVar(&billingsListFrom, "from", "", "from date (YYYY-MM-DD)")
	billingsListCmd.Flags().StringVar(&billingsListTo, "to", "", "to date (YYYY-MM-DD)")
	billingsListCmd.Flags().StringVar(&billingsListQuery, "query", "", "search query")
	billingsListCmd.Flags().BoolVar(&billingsListAll, "all", false, "fetch all pages")

	billingsCreateCmd.Flags().StringVar(&billingsCreatePartnerID, "partner-id", "", "partner ID (required)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateBillingDate, "billing-date", "", "billing date YYYY-MM-DD (required)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateDepartment, "department-id", "", "department ID (auto-resolved if omitted)")
	billingsCreateCmd.Flags().StringVar(&billingsCreateTitle, "title", "", "billing title")
	billingsCreateCmd.Flags().StringVar(&billingsCreateMemo, "memo", "", "memo")
	billingsCreateCmd.Flags().StringVar(&billingsCreatePaymentCond, "payment-condition", "", "payment condition")
	billingsCreateCmd.Flags().StringVar(&billingsCreateDueDate, "due-date", "", "due date YYYY-MM-DD")
	billingsCreateCmd.Flags().StringVar(&billingsCreateSalesDate, "sales-date", "", "sales date YYYY-MM-DD")
	billingsCreateCmd.Flags().StringArrayVar(&billingsCreateItemFlags, "item", nil, `line item: "name=X,price=N,quantity=N,excise=10"`)
	billingsCreateCmd.Flags().StringVar(&billingsCreateItemsFile, "items-file", "", "JSON or YAML file with line items")
	billingsCreateCmd.Flags().BoolVar(&billingsCreateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	billingsCreateCmd.Flags().BoolVar(&billingsCreateDryRun, "dry-run", false, "print request body without sending")
	_ = billingsCreateCmd.MarkFlagRequired("partner-id")
	_ = billingsCreateCmd.MarkFlagRequired("billing-date")

	billingsUpdateCmd.Flags().StringVar(&billingsUpdateTitle, "title", "", "billing title")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateMemo, "memo", "", "memo")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdatePaymentCond, "payment-condition", "", "payment condition")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateBillingDate, "billing-date", "", "billing date YYYY-MM-DD")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateDueDate, "due-date", "", "due date YYYY-MM-DD")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateSalesDate, "sales-date", "", "sales date YYYY-MM-DD")
	billingsUpdateCmd.Flags().StringVar(&billingsUpdateItemsFile, "items-file", "", "JSON or YAML file with line items")
	billingsUpdateCmd.Flags().BoolVar(&billingsUpdateItemsStdin, "items-stdin", false, "read line items from stdin as JSON")
	billingsUpdateCmd.Flags().BoolVar(&billingsUpdateDryRun, "dry-run", false, "print request body without sending")

	billingsDeleteCmd.Flags().BoolVar(&billingsDeleteYes, "yes", false, "skip confirmation prompt")

	billingsSetPaymentStatusCmd.Flags().StringVar(&billingsSetStatusValue, "status", "", "payment status (unsettled|settled) (required)")
	_ = billingsSetPaymentStatusCmd.MarkFlagRequired("status")

	billingsPDFCmd.Flags().BoolVar(&billingsPDFDownload, "download", false, "download PDF file")
	billingsPDFCmd.Flags().StringVar(&billingsPDFOutput, "output", "", "output file path")

	billingsCmd.AddCommand(billingsListCmd)
	billingsCmd.AddCommand(billingsShowCmd)
	billingsCmd.AddCommand(billingsCreateCmd)
	billingsCmd.AddCommand(billingsUpdateCmd)
	billingsCmd.AddCommand(billingsDeleteCmd)
	billingsCmd.AddCommand(billingsSetPaymentStatusCmd)
	billingsCmd.AddCommand(billingsPDFCmd)
}

// resolvePartnerID resolves --partner name to partner_id via search.
func resolvePartnerID(svc *api.InvoiceService, name string) (string, error) {
	partners, _, err := svc.ListPartners(pagination.Params{Page: 1, PerPage: 100}, name)
	if err != nil {
		return "", fmt.Errorf("searching partner %q: %w", name, err)
	}
	if len(partners) == 0 {
		return "", fmt.Errorf("no partner found matching %q", name)
	}
	if len(partners) > 1 {
		return "", fmt.Errorf("multiple partners found matching %q (%d results); use --partner-id instead", name, len(partners))
	}
	return partners[0].ID, nil
}

func runBillingsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve --partner name to partner_id.
	partnerID := billingsListPartnerID
	if billingsListPartner != "" {
		partnerID, err = resolvePartnerID(svc, billingsListPartner)
		if err != nil {
			return err
		}
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if billingsListAll {
		allBillings, err := fetchAll(func(page int) ([]model.Billing, *pagination.Result, error) {
			opts := api.BillingListOptions{
				Params:        pagination.Params{Page: page, PerPage: 100},
				PartnerID:     partnerID,
				PaymentStatus: billingsListPaymentStatus,
				From:          billingsListFrom,
				To:            billingsListTo,
				Query:         billingsListQuery,
			}
			return svc.ListBillings(opts)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allBillings})
		}
		return f.Format(os.Stdout, allBillings)
	}

	opts := api.BillingListOptions{
		Params:        pagination.Params{Page: billingsListPage, PerPage: billingsListPerPage},
		PartnerID:     partnerID,
		PaymentStatus: billingsListPaymentStatus,
		From:          billingsListFrom,
		To:            billingsListTo,
		Query:         billingsListQuery,
	}
	billings, pg, err := svc.ListBillings(opts)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": billings, "pagination": pg})
	}
	return f.Format(os.Stdout, billings)
}

func runBillingsShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.GetBilling(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	// Resolve line items.
	items, err := resolveLineItems(billingsCreateItemFlags, billingsCreateItemsFile, billingsCreateItemsStdin)
	if err != nil {
		return err
	}

	// Auto-resolve department_id if not provided.
	departmentID := billingsCreateDepartment
	if departmentID == "" {
		depts, err := svc.ListPartnerDepartments(billingsCreatePartnerID)
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

	params := model.CreateBillingParams{
		DepartmentID:     departmentID,
		BillingDate:      billingsCreateBillingDate,
		Title:            billingsCreateTitle,
		Memo:             billingsCreateMemo,
		PaymentCondition: billingsCreatePaymentCond,
		DueDate:          billingsCreateDueDate,
		SalesDate:        billingsCreateSalesDate,
		Items:            items,
	}

	if billingsCreateDryRun {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(params)
	}

	billing, err := svc.CreateBilling(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateBillingParams
	if cmd.Flags().Changed("title") {
		params.Title = &billingsUpdateTitle
	}
	if cmd.Flags().Changed("memo") {
		params.Memo = &billingsUpdateMemo
	}
	if cmd.Flags().Changed("payment-condition") {
		params.PaymentCondition = &billingsUpdatePaymentCond
	}
	if cmd.Flags().Changed("billing-date") {
		params.BillingDate = &billingsUpdateBillingDate
	}
	if cmd.Flags().Changed("due-date") {
		params.DueDate = &billingsUpdateDueDate
	}
	if cmd.Flags().Changed("sales-date") {
		params.SalesDate = &billingsUpdateSalesDate
	}

	// Resolve line items for update.
	items, err := resolveLineItems(nil, billingsUpdateItemsFile, billingsUpdateItemsStdin)
	if err != nil {
		return err
	}
	if items != nil {
		params.Items = items
	}

	if billingsUpdateDryRun {
		wrapped := map[string]any{"billing": params}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(wrapped)
	}

	billing, err := svc.UpdateBilling(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsDelete(cmd *cobra.Command, args []string) error {
	if !billingsDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete billing %s?", args[0]))
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

	return svc.DeleteBilling(args[0])
}

func runBillingsSetPaymentStatus(cmd *cobra.Command, args []string) error {
	switch billingsSetStatusValue {
	case "unsettled", "settled":
	default:
		return fmt.Errorf("invalid payment status %q: must be unsettled or settled", billingsSetStatusValue)
	}

	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	billing, err := svc.SetPaymentStatus(args[0], model.PaymentStatus(billingsSetStatusValue))
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, billing)
}

func runBillingsPDF(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	pdfURL, err := svc.GetBillingPDF(args[0])
	if err != nil {
		return err
	}
	if pdfURL == "" {
		return fmt.Errorf("billing %s has no PDF URL available", args[0])
	}

	if !billingsPDFDownload && billingsPDFOutput == "" {
		fmt.Println(pdfURL)
		return nil
	}

	// Download PDF using authenticated client.
	resp, err := svc.DownloadPDF(pdfURL)
	if err != nil {
		return fmt.Errorf("downloading PDF: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading PDF: unexpected status %d", resp.StatusCode)
	}

	outPath := billingsPDFOutput
	if outPath == "" {
		// Use billing ID as fallback filename.
		outPath = args[0] + ".pdf"
	}

	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("writing PDF: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Downloaded to %s\n", outPath)
	return nil
}
