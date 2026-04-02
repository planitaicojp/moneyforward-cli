package expense

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	txListPage int
	txListAll  bool

	txCreateRemark        string
	txCreateDate          string
	txCreateValue         float64
	txCreateExItemID      string
	txCreateMemo          string
	txCreateReportNumber  string
	txCreateDeptID        string
	txCreateProjectID     string
	txCreateDrExciseID    string
	txCreateCrItemID      string
	txCreateCrSubItemID   string
	txCreateJPYRate       float64
	txCreateCurrency      string

	txUpdateRemark        string
	txUpdateDate          string
	txUpdateValue         float64
	txUpdateExItemID      string
	txUpdateMemo          string
	txUpdateReportNumber  string
	txUpdateDeptID        string
	txUpdateProjectID     string
	txUpdateDrExciseID    string
	txUpdateCrItemID      string
	txUpdateCrSubItemID   string
	txUpdateJPYRate       float64
	txUpdateCurrency      string
)

var transactionsCmd = &cobra.Command{
	Use:   "transactions",
	Short: "Expense transaction operations",
}

var txListCmd = &cobra.Command{
	Use:   "list",
	Short: "List expense transactions",
	RunE:  runTxList,
}

var txShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show transaction details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runTxShow,
}

var txCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an expense transaction",
	RunE:  runTxCreate,
}

var txUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an expense transaction",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runTxUpdate,
}

var txDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an expense transaction",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runTxDelete,
}

func init() {
	// list flags
	txListCmd.Flags().IntVar(&txListPage, "page", 1, "page number")
	txListCmd.Flags().BoolVar(&txListAll, "all", false, "fetch all pages")

	// scope flag on subcommands that support it (create is personal-only)
	for _, cmd := range []*cobra.Command{txListCmd, txShowCmd, txUpdateCmd, txDeleteCmd} {
		cmd.Flags().String("scope", "personal", `scope: "personal" or "org" (admin)`)
	}

	// create flags
	txCreateCmd.Flags().StringVar(&txCreateRemark, "remark", "", "payee / description (required)")
	txCreateCmd.Flags().StringVar(&txCreateDate, "date", "", "recognized date YYYY-MM-DD (required)")
	txCreateCmd.Flags().Float64Var(&txCreateValue, "value", 0, "amount including tax (required)")
	txCreateCmd.Flags().StringVar(&txCreateExItemID, "ex-item-id", "", "expense category ID (required)")
	txCreateCmd.Flags().StringVar(&txCreateMemo, "memo", "", "memo")
	txCreateCmd.Flags().StringVar(&txCreateReportNumber, "report-number", "", "report number")
	txCreateCmd.Flags().StringVar(&txCreateDeptID, "dept-id", "", "department ID")
	txCreateCmd.Flags().StringVar(&txCreateProjectID, "project-id", "", "project ID")
	txCreateCmd.Flags().StringVar(&txCreateDrExciseID, "dr-excise-id", "", "debit tax classification ID")
	txCreateCmd.Flags().StringVar(&txCreateCrItemID, "cr-item-id", "", "credit account item ID")
	txCreateCmd.Flags().StringVar(&txCreateCrSubItemID, "cr-sub-item-id", "", "credit sub-account item ID")
	txCreateCmd.Flags().Float64Var(&txCreateJPYRate, "jpyrate", 0, "JPY exchange rate")
	txCreateCmd.Flags().StringVar(&txCreateCurrency, "currency", "", "currency code (e.g. USD)")
	_ = txCreateCmd.MarkFlagRequired("remark")
	_ = txCreateCmd.MarkFlagRequired("date")
	_ = txCreateCmd.MarkFlagRequired("value")
	_ = txCreateCmd.MarkFlagRequired("ex-item-id")

	// update flags
	txUpdateCmd.Flags().StringVar(&txUpdateRemark, "remark", "", "payee / description")
	txUpdateCmd.Flags().StringVar(&txUpdateDate, "date", "", "recognized date YYYY-MM-DD")
	txUpdateCmd.Flags().Float64Var(&txUpdateValue, "value", 0, "amount including tax")
	txUpdateCmd.Flags().StringVar(&txUpdateExItemID, "ex-item-id", "", "expense category ID")
	txUpdateCmd.Flags().StringVar(&txUpdateMemo, "memo", "", "memo")
	txUpdateCmd.Flags().StringVar(&txUpdateReportNumber, "report-number", "", "report number")
	txUpdateCmd.Flags().StringVar(&txUpdateDeptID, "dept-id", "", "department ID")
	txUpdateCmd.Flags().StringVar(&txUpdateProjectID, "project-id", "", "project ID")
	txUpdateCmd.Flags().StringVar(&txUpdateDrExciseID, "dr-excise-id", "", "debit tax classification ID")
	txUpdateCmd.Flags().StringVar(&txUpdateCrItemID, "cr-item-id", "", "credit account item ID")
	txUpdateCmd.Flags().StringVar(&txUpdateCrSubItemID, "cr-sub-item-id", "", "credit sub-account item ID")
	txUpdateCmd.Flags().Float64Var(&txUpdateJPYRate, "jpyrate", 0, "JPY exchange rate")
	txUpdateCmd.Flags().StringVar(&txUpdateCurrency, "currency", "", "currency code (e.g. USD)")

	transactionsCmd.AddCommand(txListCmd, txShowCmd, txCreateCmd, txUpdateCmd, txDeleteCmd)
}

func isOrgScope(cmd *cobra.Command) bool {
	scope, _ := cmd.Flags().GetString("scope")
	if scope != "" && scope != "personal" && scope != "org" {
		// Invalid value will be caught by the caller; default to personal.
		return false
	}
	return scope == "org"
}

func validateScope(cmd *cobra.Command) error {
	scope, _ := cmd.Flags().GetString("scope")
	if scope != "personal" && scope != "org" {
		return fmt.Errorf("invalid --scope value %q: must be \"personal\" or \"org\"", scope)
	}
	return nil
}

func runTxList(cmd *cobra.Command, args []string) error {
	if err := validateScope(cmd); err != nil {
		return err
	}
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

	listFn := svc.ListMyTransactions
	if isOrgScope(cmd) {
		listFn = svc.ListOrgTransactions
	}

	if txListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExTransaction, bool, error) {
			return listFn(oid, page)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"ex_transactions": all})
		}
		return f.Format(os.Stdout, all)
	}

	txns, _, err := listFn(oid, txListPage)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"ex_transactions": txns})
	}
	return f.Format(os.Stdout, txns)
}

func runTxShow(cmd *cobra.Command, args []string) error {
	if err := validateScope(cmd); err != nil {
		return err
	}
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	var tx *model.ExTransaction
	if isOrgScope(cmd) {
		tx, err = svc.GetOrgTransaction(oid, args[0])
	} else {
		tx, err = svc.GetMyTransaction(oid, args[0])
	}
	if err != nil {
		return err
	}
	return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, tx)
}

func runTxCreate(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	input := model.ExTransactionCreateInput{
		Remark:       txCreateRemark,
		RecognizedAt: txCreateDate,
		Value:        txCreateValue,
		ExItemID:     txCreateExItemID,
		Memo:         txCreateMemo,
		ReportNumber: txCreateReportNumber,
		DeptID:       txCreateDeptID,
		ProjectID:    txCreateProjectID,
		DrExciseID:   txCreateDrExciseID,
		CrItemID:     txCreateCrItemID,
		CrSubItemID:  txCreateCrSubItemID,
		JPYRate:      txCreateJPYRate,
		Currency:     txCreateCurrency,
	}

	tx, err := svc.CreateMyTransaction(oid, input)
	if err != nil {
		return err
	}
	return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, tx)
}

func runTxUpdate(cmd *cobra.Command, args []string) error {
	if err := validateScope(cmd); err != nil {
		return err
	}
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	var input model.ExTransactionUpdateInput
	if cmd.Flags().Changed("remark") {
		input.Remark = &txUpdateRemark
	}
	if cmd.Flags().Changed("date") {
		input.RecognizedAt = &txUpdateDate
	}
	if cmd.Flags().Changed("value") {
		input.Value = &txUpdateValue
	}
	if cmd.Flags().Changed("ex-item-id") {
		input.ExItemID = &txUpdateExItemID
	}
	if cmd.Flags().Changed("memo") {
		input.Memo = &txUpdateMemo
	}
	if cmd.Flags().Changed("report-number") {
		input.ReportNumber = &txUpdateReportNumber
	}
	if cmd.Flags().Changed("dr-excise-id") {
		input.DrExciseID = &txUpdateDrExciseID
	}
	if cmd.Flags().Changed("dept-id") {
		input.DeptID = &txUpdateDeptID
	}
	if cmd.Flags().Changed("project-id") {
		input.ProjectID = &txUpdateProjectID
	}
	if cmd.Flags().Changed("cr-item-id") {
		input.CrItemID = &txUpdateCrItemID
	}
	if cmd.Flags().Changed("cr-sub-item-id") {
		input.CrSubItemID = &txUpdateCrSubItemID
	}
	if cmd.Flags().Changed("jpyrate") {
		input.JPYRate = &txUpdateJPYRate
	}
	if cmd.Flags().Changed("currency") {
		input.Currency = &txUpdateCurrency
	}

	var tx *model.ExTransaction
	if isOrgScope(cmd) {
		tx, err = svc.UpdateOrgTransaction(oid, args[0], input)
	} else {
		tx, err = svc.UpdateMyTransaction(oid, args[0], input)
	}
	if err != nil {
		return err
	}
	return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, tx)
}

func runTxDelete(cmd *cobra.Command, args []string) error {
	if err := validateScope(cmd); err != nil {
		return err
	}
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}

	if isOrgScope(cmd) {
		err = svc.DeleteOrgTransaction(oid, args[0])
	} else {
		err = svc.DeleteMyTransaction(oid, args[0])
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Deleted transaction %s\n", args[0])
	return nil
}
