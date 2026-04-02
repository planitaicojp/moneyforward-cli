package expense

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/planitaicojp/moneyforward-cli/cmd/cmdutil"
	"github.com/planitaicojp/moneyforward-cli/internal/model"
	"github.com/planitaicojp/moneyforward-cli/internal/output"
)

var (
	projectsListPage    int
	projectsListAll     bool
	projectsListKeyword string
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "Project operations",
}

var projectsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	RunE:  runProjectsList,
}

var projectsShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show project details",
	Args:  cmdutil.ExactArgs(1),
	RunE:  runProjectsShow,
}

func init() {
	projectsListCmd.Flags().IntVar(&projectsListPage, "page", 1, "page number")
	projectsListCmd.Flags().BoolVar(&projectsListAll, "all", false, "fetch all pages")
	projectsListCmd.Flags().StringVar(&projectsListKeyword, "keyword", "", "search keyword (max 50 chars)")
	projectsCmd.AddCommand(projectsListCmd)
	projectsCmd.AddCommand(projectsShowCmd)
}

func runProjectsList(cmd *cobra.Command, args []string) error {
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

	if projectsListAll {
		all, err := cmdutil.FetchAllExpense(func(page int) ([]model.ExpenseProject, bool, error) {
			return svc.ListProjects(oid, page, projectsListKeyword)
		})
		if err != nil {
			return err
		}
		if format == "json" {
			return f.Format(os.Stdout, map[string]any{"projects": all})
		}
		return f.Format(os.Stdout, all)
	}

	projects, _, err := svc.ListProjects(oid, projectsListPage, projectsListKeyword)
	if err != nil {
		return err
	}
	if format == "json" {
		return f.Format(os.Stdout, map[string]any{"projects": projects})
	}
	return f.Format(os.Stdout, projects)
}

func runProjectsShow(cmd *cobra.Command, args []string) error {
	svc, err := newExpenseService(cmd)
	if err != nil {
		return err
	}
	oid, err := resolveOfficeID(cmd, svc)
	if err != nil {
		return err
	}
	project, err := svc.GetProject(oid, args[0])
	if err != nil {
		return err
	}
	f := output.New(cmdutil.GetFormat(cmd))
	return f.Format(os.Stdout, project)
}
