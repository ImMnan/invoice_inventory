package update

import (
	"github.com/spf13/cobra"
)

// applyCMD represents the get command
var UpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Use update command for updating the resources in the Database",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
}
