package auth

import "github.com/spf13/cobra"

// AuthCmd is the parent auth command.
var AuthCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Money Forward Cloud authentication",
	Long:  "Log in, log out, and inspect authentication status for Money Forward Cloud services.",
}

func init() {
	AuthCmd.AddCommand(loginCmd)
	AuthCmd.AddCommand(logoutCmd)
	AuthCmd.AddCommand(statusCmd)
	AuthCmd.AddCommand(listCmd)
	AuthCmd.AddCommand(switchCmd)
	AuthCmd.AddCommand(removeCmd)
	AuthCmd.AddCommand(tokenCmd)
}
