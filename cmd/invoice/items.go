package invoice

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
	"github.com/planitaicojp/moneyforward-cli/internal/pagination"
	"github.com/planitaicojp/moneyforward-cli/internal/prompt"
)

var itemsCmd = &cobra.Command{
	Use:   "items",
	Short: "Item operations",
}

// --- list ---

var (
	itemsListPage    int
	itemsListPerPage int
	itemsListQuery   string
	itemsListAll     bool
)

var itemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items",
	RunE:  runItemsList,
}

// --- show ---

var itemsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show item details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsShow,
}

// --- create ---

var (
	itemsCreateName     string
	itemsCreateCode     string
	itemsCreateDetail   string
	itemsCreateUnit     string
	itemsCreatePrice    int
	itemsCreateQuantity int
	itemsCreateExcise   string
)

var itemsCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an item",
	RunE:  runItemsCreate,
}

// --- update ---

var (
	itemsUpdateName     string
	itemsUpdateCode     string
	itemsUpdateDetail   string
	itemsUpdateUnit     string
	itemsUpdatePrice    int
	itemsUpdateQuantity int
	itemsUpdateExcise   string
)

var itemsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an item",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsUpdate,
}

// --- delete ---

var itemsDeleteYes bool

var itemsDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an item",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runItemsDelete,
}

func init() {
	itemsListCmd.Flags().IntVar(&itemsListPage, "page", 1, "page number")
	itemsListCmd.Flags().IntVar(&itemsListPerPage, "per-page", 25, "items per page (max 100)")
	itemsListCmd.Flags().StringVar(&itemsListQuery, "query", "", "search query")
	itemsListCmd.Flags().BoolVar(&itemsListAll, "all", false, "fetch all pages")

	itemsCreateCmd.Flags().StringVar(&itemsCreateName, "name", "", "item name (required)")
	itemsCreateCmd.Flags().StringVar(&itemsCreateCode, "code", "", "item code")
	itemsCreateCmd.Flags().StringVar(&itemsCreateDetail, "detail", "", "item detail")
	itemsCreateCmd.Flags().StringVar(&itemsCreateUnit, "unit", "", "unit (e.g. hours, pcs)")
	itemsCreateCmd.Flags().IntVar(&itemsCreatePrice, "price", 0, "unit price")
	itemsCreateCmd.Flags().IntVar(&itemsCreateQuantity, "quantity", 0, "quantity")
	itemsCreateCmd.Flags().StringVar(&itemsCreateExcise, "excise", "", "excise type (10, 8, 8r, 5, 0, exempt, non)")
	_ = itemsCreateCmd.MarkFlagRequired("name")

	itemsUpdateCmd.Flags().StringVar(&itemsUpdateName, "name", "", "item name")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateCode, "code", "", "item code")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateDetail, "detail", "", "item detail")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateUnit, "unit", "", "unit")
	itemsUpdateCmd.Flags().IntVar(&itemsUpdatePrice, "price", 0, "unit price")
	itemsUpdateCmd.Flags().IntVar(&itemsUpdateQuantity, "quantity", 0, "quantity")
	itemsUpdateCmd.Flags().StringVar(&itemsUpdateExcise, "excise", "", "excise type")

	itemsDeleteCmd.Flags().BoolVar(&itemsDeleteYes, "yes", false, "skip confirmation prompt")

	itemsCmd.AddCommand(itemsListCmd)
	itemsCmd.AddCommand(itemsShowCmd)
	itemsCmd.AddCommand(itemsCreateCmd)
	itemsCmd.AddCommand(itemsUpdateCmd)
	itemsCmd.AddCommand(itemsDeleteCmd)
}

func runItemsList(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	format := cmdutil.GetFormat(cmd)
	f := output.New(format)

	if itemsListAll {
		allItems, err := fetchAll(func(page int) ([]model.Item, *pagination.Result, error) {
			return svc.ListItems(pagination.Params{Page: page, PerPage: 100}, itemsListQuery)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"data": allItems})
		}
		return f.Format(os.Stdout, allItems)
	}

	params := pagination.Params{Page: itemsListPage, PerPage: itemsListPerPage}
	items, pg, err := svc.ListItems(params, itemsListQuery)
	if err != nil {
		return err
	}

	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"data": items, "pagination": pg})
	}
	return f.Format(os.Stdout, items)
}

func runItemsShow(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	item, err := svc.GetItem(args[0])
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsCreate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	params := model.CreateItemParams{
		Name:   itemsCreateName,
		Code:   itemsCreateCode,
		Detail: itemsCreateDetail,
		Unit:   itemsCreateUnit,
	}
	if cmd.Flags().Changed("price") {
		params.Price = &itemsCreatePrice
	}
	if cmd.Flags().Changed("quantity") {
		params.Quantity = &itemsCreateQuantity
	}
	if itemsCreateExcise != "" {
		excise := resolveExcise(itemsCreateExcise)
		params.Excise = excise
	}

	item, err := svc.CreateItem(params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsUpdate(cmd *cobra.Command, args []string) error {
	svc, err := newInvoiceService(cmd)
	if err != nil {
		return err
	}

	var params model.UpdateItemParams
	if cmd.Flags().Changed("name") {
		params.Name = &itemsUpdateName
	}
	if cmd.Flags().Changed("code") {
		params.Code = &itemsUpdateCode
	}
	if cmd.Flags().Changed("detail") {
		params.Detail = &itemsUpdateDetail
	}
	if cmd.Flags().Changed("unit") {
		params.Unit = &itemsUpdateUnit
	}
	if cmd.Flags().Changed("price") {
		params.Price = &itemsUpdatePrice
	}
	if cmd.Flags().Changed("quantity") {
		params.Quantity = &itemsUpdateQuantity
	}
	if cmd.Flags().Changed("excise") {
		excise := resolveExcise(itemsUpdateExcise)
		params.Excise = &excise
	}

	item, err := svc.UpdateItem(args[0], params)
	if err != nil {
		return err
	}

	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, item)
}

func runItemsDelete(cmd *cobra.Command, args []string) error {
	if !itemsDeleteYes {
		ok, err := prompt.Confirm(fmt.Sprintf("Delete item %s?", args[0]))
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

	return svc.DeleteItem(args[0])
}
