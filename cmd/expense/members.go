package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	membersListPage       int
	membersListAll        bool
	membersListOnlyActive bool
)

var membersCmd = &cobra.Command{
	Use:   "members",
	Short: "Office member operations (v2)",
}

var membersListCmd = &cobra.Command{
	Use:   "list",
	Short: "List office members",
	RunE:  runMembersList,
}

var membersShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show member details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runMembersShow,
}

var membersMeCmd = &cobra.Command{
	Use:   "me",
	Short: "Show authenticated user's member info",
	RunE:  runMembersMe,
}

func init() {
	membersListCmd.Flags().IntVar(&membersListPage, "page", 1, "page number")
	membersListCmd.Flags().BoolVar(&membersListAll, "all", false, "fetch all pages")
	membersListCmd.Flags().BoolVar(&membersListOnlyActive, "only-active", false, "only active members")
	membersCmd.AddCommand(membersListCmd)
	membersCmd.AddCommand(membersShowCmd)
	membersCmd.AddCommand(membersMeCmd)
}

func runMembersList(cmd *cobra.Command, args []string) error {
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

	if membersListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.OfficeMemberV2, bool, error) {
			return svc.ListMembersV2(oid, page, membersListOnlyActive)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"office_members": all})
		}
		return f.Format(os.Stdout, all)
	}

	members, _, err := svc.ListMembersV2(oid, membersListPage, membersListOnlyActive)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"office_members": members})
	}
	return f.Format(os.Stdout, members)
}

func runMembersShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	member, err := svc.GetMemberV2(oid, args[0])
	if err != nil {
		return err
	}
	return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, member)
}

func runMembersMe(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	me, err := svc.GetMe(oid)
	if err != nil {
		return err
	}
	return output.New(cmdutil.GetFormat(cmd)).Format(os.Stdout, me)
}
